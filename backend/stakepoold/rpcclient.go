package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/vatercoin/vaterd/chaincfg/chainhash"
	"github.com/vatercoin/vaterd/rpcclient"
	"github.com/vatercoin/vaterstakepool/backend/stakepoold/userdata"
)

var requiredChainServerAPI = semver{major: 5, minor: 0, patch: 0}
var requiredWalletAPI = semver{major: 5, minor: 0, patch: 0}

func connectNodeRPC(ctx *appContext, cfg *config) (*rpcclient.Client, semver, error) {
	var nodeVer semver

	vaterdCert, err := ioutil.ReadFile(cfg.VaterdCert)
	if err != nil {
		log.Errorf("Failed to read vaterd cert file at %s: %s\n",
			cfg.VaterdCert, err.Error())
		return nil, nodeVer, err
	}

	log.Debugf("Attempting to connect to vaterd RPC %s as user %s "+
		"using certificate located in %s",
		cfg.VaterdHost, cfg.VaterdUser, cfg.VaterdCert)

	connCfgDaemon := &rpcclient.ConnConfig{
		Host:         cfg.VaterdHost,
		Endpoint:     "ws", // websocket
		User:         cfg.VaterdUser,
		Pass:         cfg.VaterdPassword,
		Certificates: vaterdCert,
	}

	ntfnHandlers := getNodeNtfnHandlers(ctx)
	vaterdClient, err := rpcclient.New(connCfgDaemon, ntfnHandlers)
	if err != nil {
		log.Errorf("Failed to start vaterd RPC client: %s\n", err.Error())
		return nil, nodeVer, err
	}

	// Ensure the RPC server has a compatible API version.
	ver, err := vaterdClient.Version()
	if err != nil {
		log.Error("Unable to get RPC version: ", err)
		return nil, nodeVer, fmt.Errorf("Unable to get node RPC version")
	}

	vaterdVer := ver["vaterdjsonrpcapi"]
	nodeVer = semver{vaterdVer.Major, vaterdVer.Minor, vaterdVer.Patch}

	if !semverCompatible(requiredChainServerAPI, nodeVer) {
		return nil, nodeVer, fmt.Errorf("Node JSON-RPC server does not have "+
			"a compatible API version. Advertises %v but require %v",
			nodeVer, requiredChainServerAPI)
	}

	return vaterdClient, nodeVer, nil
}

func connectWalletRPC(cfg *config) (*rpcclient.Client, semver, error) {
	var walletVer semver

	vaterwCert, err := ioutil.ReadFile(cfg.WalletCert)
	if err != nil {
		log.Errorf("Failed to read vaterwallet cert file at %s: %s\n",
			cfg.WalletCert, err.Error())
		return nil, walletVer, err
	}

	log.Infof("Attempting to connect to vaterwallet RPC %s as user %s "+
		"using certificate located in %s",
		cfg.WalletHost, cfg.WalletUser, cfg.WalletCert)

	connCfgWallet := &rpcclient.ConnConfig{
		Host:         cfg.WalletHost,
		Endpoint:     "ws",
		User:         cfg.WalletUser,
		Pass:         cfg.WalletPassword,
		Certificates: vaterwCert,
	}

	ntfnHandlers := getWalletNtfnHandlers()
	vaterwClient, err := rpcclient.New(connCfgWallet, ntfnHandlers)
	if err != nil {
		log.Errorf("Verify that username and password is correct and that "+
			"rpc.cert is for your wallet: %v", cfg.WalletCert)
		return nil, walletVer, err
	}

	// Ensure the wallet RPC server has a compatible API version.
	ver, err := vaterwClient.Version()
	if err != nil {
		log.Error("Unable to get RPC version: ", err)
		return nil, walletVer, fmt.Errorf("Unable to get node RPC version")
	}

	vaterwVer := ver["vaterwalletjsonrpcapi"]
	walletVer = semver{vaterwVer.Major, vaterwVer.Minor, vaterwVer.Patch}

	if !semverCompatible(requiredWalletAPI, walletVer) {
		log.Warnf("Node JSON-RPC server %v does not have "+
			"a compatible API version. Advertizes %v but require %v",
			cfg.WalletHost, walletVer, requiredWalletAPI)
	}

	return vaterwClient, walletVer, nil
}

func walletGetTickets(ctx *appContext) (map[chainhash.Hash]string, map[chainhash.Hash]string, error) {
	blockHashToHeightCache := make(map[chainhash.Hash]int32)

	// This is suboptimal to copy and needs fixing.
	userVotingConfig := make(map[string]userdata.UserVotingConfig)
	ctx.RLock()
	for k, v := range ctx.userVotingConfig {
		userVotingConfig[k] = v
	}
	ctx.RUnlock()

	ignoredLowFeeTickets := make(map[chainhash.Hash]string)
	liveTickets := make(map[chainhash.Hash]string)
	normalFee := 0

	log.Info("Calling GetTickets...")
	timenow := time.Now()
	tickets, err := ctx.walletConnection.GetTickets(false)
	log.Infof("GetTickets: took %v", time.Since(timenow))

	if err != nil {
		log.Warnf("GetTickets failed: %v", err)
		return ignoredLowFeeTickets, liveTickets, err
	}

	type promise struct {
		rpcclient.FutureGetTransactionResult
	}
	promises := make([]promise, 0, len(tickets))

	log.Debugf("setting up GetTransactionAsync for %v tickets", len(tickets))
	for _, ticket := range tickets {
		// lookup ownership of each ticket
		promises = append(promises, promise{ctx.walletConnection.GetTransactionAsync(ticket)})
	}

	counter := 0
	for _, p := range promises {
		counter++
		log.Debugf("Receiving GetTransaction result for ticket %v/%v", counter, len(tickets))
		gt, err := p.Receive()
		if err != nil {
			// All tickets should exist and be able to be looked up
			log.Warnf("GetTransaction error: %v", err)
			continue
		}
		for i := range gt.Details {
			addr := gt.Details[i].Address
			_, ok := userVotingConfig[addr]
			if !ok {
				log.Warnf("Could not map ticket %v to a user, user %v doesn't exist", gt.TxID, addr)
				continue
			}

			hash, err := chainhash.NewHashFromStr(gt.TxID)
			if err != nil {
				log.Warnf("invalid ticket %v", err)
				continue
			}

			// All tickets are present in the GetTickets response, whether they
			// pay the correct fee or not.  So we need to verify fees and
			// sort the tickets into their respective maps.
			_, isAdded := ctx.addedLowFeeTicketsMSA[*hash]
			if isAdded {
				liveTickets[*hash] = userVotingConfig[addr].MultiSigAddress
			} else {

				msgTx, err := MsgTxFromHex(gt.Hex)
				if err != nil {
					log.Warnf("MsgTxFromHex failed for %v: %v", gt.Hex, err)
					continue
				}

				// look up the height at which this ticket was purchased
				var ticketBlockHeight int32
				ticketBlockHash, err := chainhash.NewHashFromStr(gt.BlockHash)
				if err != nil {
					log.Warnf("NewHashFromStr failed for %v: %v", gt.BlockHash, err)
					continue
				}

				height, inCache := blockHashToHeightCache[*ticketBlockHash]
				if inCache {
					ticketBlockHeight = height
				} else {
					gbh, err := ctx.nodeConnection.GetBlockHeader(ticketBlockHash)
					if err != nil {
						log.Warnf("GetBlockHeader failed for %v: %v", ticketBlockHash, err)
						continue
					}

					blockHashToHeightCache[*ticketBlockHash] = int32(gbh.Height)
					ticketBlockHeight = int32(gbh.Height)
				}

				ticketFeesValid, err := evaluateStakePoolTicket(ctx, msgTx, ticketBlockHeight)
				if ticketFeesValid {
					normalFee++
					liveTickets[*hash] = userVotingConfig[addr].MultiSigAddress
				} else {
					ignoredLowFeeTickets[*hash] = userVotingConfig[addr].MultiSigAddress
					log.Warnf("ignoring ticket %v for msa %v ticketFeesValid %v err %v",
						*hash, ctx.userVotingConfig[addr].MultiSigAddress, ticketFeesValid, err)
				}
			}
			break
		}
	}

	log.Infof("tickets loaded -- addedLowFee %v ignoredLowFee %v normalFee %v "+
		"live %v total %v", len(ctx.addedLowFeeTicketsMSA),
		len(ignoredLowFeeTickets), normalFee, len(liveTickets),
		len(tickets))

	return ignoredLowFeeTickets, liveTickets, nil
}
