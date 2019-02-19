# vaterstakepool

[![GoDoc](https://godoc.org/github.com/vatercoin/vaterstakepool?status.svg)](https://godoc.org/github.com/vatercoin/vaterstakepool)
[![Build Status](https://travis-ci.org/vatercoin/vaterstakepool.svg?branch=master)](https://travis-ci.org/vatercoin/vaterstakepool)
[![Go Report Card](https://goreportcard.com/badge/github.com/vatercoin/vaterstakepool)](https://goreportcard.com/report/github.com/vatercoin/vaterstakepool)

vaterstakepool is a web application which coordinates generating 1-of-2 multisig
addresses on a pool of [vaterwallet](https://github.com/vatercoin/vaterwallet) servers
so users can purchase [proof-of-stake tickets](https://docs.vatercoin.org/mining/proof-of-stake/)
on the [Vatercoin](https://vatercoin.org/) network and have the pool of wallet servers
vote on their behalf when the ticket is selected.

## Architecture

![Voting Service Architecture](https://i.imgur.com/2JDA9dl.png)

- It is highly recommended to use 3 vaterd+vaterwallet+stakepoold nodes for
  production use on mainnet.
- The architecture is subject to change in the future to lessen the dependence
  on vaterwallet and MySQL.

## Git Tip Release notes

- The handling of tickets considered invalid because they pay too-low-of-a-fee
  is now integrated directly into vaterstakepool and stakepoold.
  - Users who pass both the adminIPs and the new adminUserIDs checks will see a
    new link on the menu to the new administrative add tickets page.
  - Tickets are added to the MySQL database and then stakepoold is triggered to
    pull an update from the database and reload its config.
  - To accommodate changes to the gRPC API, vaterstakepool/stakepoold had their
    API versions changed to require/advertize 4.0.0. This requires performing
    the upgrade steps outlined below.
- **KNOWN ISSUE** Total tickets count reported by stakepoold may not be totally
  accurate until low fee tickets that have been added to the database can be
  marked as voted.  This will be resolved by future work.
  ([#201](https://github.com/vatercoin/vaterstakepool/issues/201)).

## Git Tip Upgrade Guide

1) Announce maintenance and shut down vaterstakepool.
2) Upgrade Go to the latest stable version if necessary/possible.
3) Perform an upgrade of each stakepoold instance one at a time.
   - Stop stakepoold.
   - Build and restart stakepoold.
4) Edit vaterstakepool.conf and set adminIPs/adminUserIDs appropriately to include
   the administrative staff for whom you wish give the ability to add low fee
   tickets for voting.
5) Upgrade and start vaterstakepool after setting adminUserIDs.
6) Announce maintenance complete after verifying functionality.

## 1.1.1 Release Notes

- vaterd has a new agenda and the vote version in vaterwallet has been
  incremented to v5 on mainnet.
- stakepoold
  - The ticket list is now maintained by doing an initial GetTicket RPC call and
    then subtracts/adds tickets by processing SpentAndMissed/New ticket
    notifications from vaterwallet.  This approach is much faster than the old
    method of calling StakePoolUserInfo for each user.
  - Bug fixes to the above commit and to accommodate changes in vaterwallet.
- Status page
  - StatusUnauthorized error is now thrown rather than a generic one when
    accessing the page as a non-admin.
  - Updated to use new design.
  - Synced vaterwallet walletinfo field list.
- Tickets page
  - Performance was greatly improved by skipping display of historic tickets.
  - Handles users that have only low fee/invalid tickets properly.
  - Expired tickets are now separated from missed.
- General markup improvements.
  - Removed mention of creating a voting account as it has been deprecated.
  - Instructions were further clarified and updated to strongly recommend the
    use of Vatercoiniton/Paymetheus.
  - Fragments of invalid markup were fixed.

## 1.1.1 Upgrade Guide

1) Announce maintenance and shut down vaterstakepool.
2) Perform upgrades on each vaterd+vaterwallet+stakepoold voting cluster one at a
   time.
   - Stop stakepoold, vaterwallet, and vaterd.
   - Upgrade vaterd, vaterwallet to 1.1.0 release binaries or git. If compiling from
   source, Go 1.9 is recommended to pick up improvements to the Golang runtime.
   - Restart vaterd, vaterwallet.
   - Upgrade stakepoold.
   - Start stakepoold.
3) Upgrade and start vaterstakepool.  If you are maintaining a fork, note that
   you need to update the vaterd/chaincfg dependency to a revision that contains
   the new agenda.
4) vaterstakepool will reset the votebits for all users to 1 when it detects the
   new vote version via stakepoold.
5) Announce maintenance complete after verifying functionality.  If possible,
   also announce that a new voting agenda is available and users must login
   to set their preferences for the new agenda.

## Requirements

- [Go](http://golang.org) 1.10.5 or newer (1.11 is recommended).
- MySQL
- Nginx or other web server to proxy to vaterstakepool

## Installation

### Build from Source

Building or updating from source requires only an installation of Go
([instructions](http://golang.org/doc/install)). It is recommended to add
`$GOPATH/bin` to your `PATH` at this point.

Clone the vaterstakepool repository into any folder and follow the instructions
below for your version of Go.

#### Building with Go 1.11

Go 1.11 introduced native support for
[modules](https://github.com/golang/go/wiki/Modules), a new dependency
management approach, that obviates the need for third party tooling such as
`dep`.

Usage is simple, and nothing is required except Go 1.11. If building in a folder
under `GOPATH`, it is necessary to explicitly build with modules enabled:

    GO111MODULE=on go build

If building outside of `GOPATH`, modules are automatically enabled, and `go
build` is sufficient.

The `go` tool will process the source code and automatically download
dependencies. If the dependencies are configured correctly, there will be no
modifications to the `go.mod` and `go.sum` files.

#### Building with Go 1.10

Module-enabled builds with Go 1.10 require the
[vgo](https://github.com/golang/vgo) command. Follow the same procedures as if
you were [using Go 1.11](#building-with-go-111), but replacing `go` with `vgo`.

**NOTE:** The `dep` tool is no longer supported. If you must use Go 1.10,
install and use `vgo`. If possible, upgrade to Go 1.11.

### Components

The frontend server (vaterstakepool) and the backend daemon (stakepoold) are built
separately. Since module-enabled builds no longer require building under
`$GOPATH`, the following instructions use the placeholder
`{{YOUR_GO_MODULE_PATH}}` to refer to wherever you checkout your Go code.

#### Frontend - vaterstakepool

Build vaterstakepool and copy it to the web server.

```bash
$ cd {{YOUR_GO_MODULE_PATH}}/github.com/vatercoin/vaterstakepool
$ go build
```

#### Backend - stakepoold

Build stakepoold and copy it to each voting wallet node.

```bash
$ cd {{YOUR_GO_MODULE_PATH}}/src/github.com/vatercoin/vaterstakepool/backend/stakepoold
$ go build
```

## Updating

To update an existing source tree, pull the latest changes and install the
matching dependencies:

```bash
$ cd $GOPATH/src/github.com/vatercoin/vaterstakepool
$ git pull
$ dep ensure
$ go build
$ cd $GOPATH/src/github.com/vatercoin/vaterstakepool/backend/stakepoold
$ go build
```

## Setup

### Pre-requisites

These instructions assume you are familiar with vaterd/vaterwallet.

- Create basic vaterd/vaterwallet/vaterctl config files with usernames, passwords,
  rpclisten, and network set appropriately within them or run example commands
  with additional flags as necessary.

- Build/install vaterd and vaterwallet from latest master.

- Run vaterd instances and let them fully sync.

### Voting service fees/cold wallet

- Setup a new wallet for receiving payment for voting service fees.  **This should
  be completely separate from the voting service infrastructure.**

```bash
$ vaterwallet --create
$ vaterwallet
```

- Get the master pubkey for the account you wish to use. This will be needed to
  configure vaterwallet and vaterstakepool.

```bash
$ vaterctl --wallet createnewaccount teststakepoolfees
$ vaterctl --wallet getmasterpubkey teststakepoolfees
```

- Mark 10000 addresses in use for the account so the wallet will recognize
  transactions to those addresses. Fees from UserId 1 will go to address 1,
  UserId 2 to address 2, and so on.

```bash
$ vaterctl --wallet accountsyncaddressindex teststakepoolfees 0 10000
```

### Voting service voting wallets

- Create the wallets.  All wallets should have the same seed.  **Backup the seed
  for disaster recovery!**

```bash
$ vaterwallet --create
```

- Start a properly configured vaterwallet and unlock it. See
  sample-vaterwallet.conf.

```bash
$ vaterwallet
```

- Get the master pubkey from the default account.  This will be used for
  votingwalletextpub in vaterstakepool.conf.

```bash
$ vaterctl --wallet getmasterpubkey default
```

### MySQL

- Install, configure, and start MySQL
- Add stakepool user and create the stakepool database

```bash
$ mysql -uroot -ppassword

MySQL> CREATE USER 'stakepool'@'localhost' IDENTIFIED BY 'password';
MySQL> GRANT ALL PRIVILEGES ON *.* TO 'stakepool'@'localhost' WITH GRANT OPTION;
MySQL> FLUSH PRIVILEGES;
MySQL> CREATE DATABASE stakepool;
```

### Nginx/web server

- Adapt sample-nginx.conf or setup a different web server in a proxy
  configuration.

### stakepoold setup

- Adapt sample-stakepoold.conf and run stakepoold.

### vaterstakepool setup

- Create the .vaterstakepool directory and copy vaterwallet certs to it:

```bash
$ mkdir ~/.vaterstakepool
$ cd ~/.vaterstakepool
$ scp walletserver1:~/.vaterwallet/rpc.cert wallet1.cert
$ scp walletserver2:~/.vaterwallet/rpc.cert wallet2.cert
$ scp walletserver1:~/.stakepoold/rpc.cert stakepoold1.cert
$ scp walletserver2:~/.stakepoold/rpc.cert stakepoold2.cert
```

- Copy sample config and edit appropriately.

```bash
$ cp -p sample-vaterstakepool.conf vaterstakepool.conf
```

## Running

The easiest way to run the stakepool code is to run it directly from the root of
the source tree:

```bash
$ cd $GOPATH/src/github.com/vatercoin/vaterstakepool
$ go build
$ ./vaterstakepool
```

If you wish to run vaterstakepool from a different directory you will need to
change **publicpath** and **templatepath** from their relative paths to an
absolute path.

## Development

If you are modifying templates, sending the USR1 signal to the vaterstakepool
process will trigger a template reload.

## Operations

- vaterstakepool will connect to the database or error out if it cannot do so.

- vaterstakepool will create the stakepool.Users table automatically if it doesn't
  exist.

- vaterstakepool attempts to connect to all of the wallet servers on startup or
  error out if it cannot do so.

- vaterstakepool takes a user's pubkey, validates it, calls getnewaddress on all
  the wallet servers, then createmultisig, and finally importscript.  If any of
  these RPCs fail or returns inconsistent results, the RPC client built-in to
  vaterstakepool will shut down and will not operate until it has been restarted.
  Wallets should be verified to be in sync before restarting.

- User API Tokens have an issuer field set to baseURL from the configuration file.
  Changing the baseURL requires all API Tokens to be re-generated.

## Adding Invalid Tickets

### For Newer versions / git tip

If a user pays an incorrect fee, login as an account that meets the
adminUserIps and adminUserIds restrictions and click the 'Add Low Fee Tickets'
link in the menu.  You will be presented with a list of tickets that are
suitable for adding.  Check the appropriate one(s) and click the submit button.
Upon success, you should see the stakepoold logs reflect that the new tickets
were processed.

### For v1.1.1 and below

If a user pays an incorrect fee you may add their tickets like so (requires vaterd
running with `txindex=1`):

```bash
vaterctl --wallet stakepooluserinfo "MultiSigAddress" | grep -Pzo '(?<="invalid": \[)[^\]]*' | tr -d , | xargs -Itickethash vaterctl --wallet getrawtransaction tickethash | xargs -Itickethex vaterctl --wallet addticket "tickethex"
```

## Backups, monitoring, security considerations

- MySQL should be backed up often and regularly (probably at least hourly).
  Backups should be transferred off-site.  If using binary backups, do a test
  restore. For .sql files, verify visually.

- A monitoring system with alerting should be pointed at vaterstakepool and
  tested/verified to be operating properly.  There is a hidden /status page
  which throws 500 if the RPC client is shutdown.  If your monitoring system
  supports it, add additional points of verification such as: checking that the
  /stats page loads and has expected information in it, create a test account
  and setup automated login testing, etc.

- Wallets should never be used for anything else (they should always have a
  balance of 0).

## Disaster Recovery

**Always keep at least one wallet voting while performing maintenance / restoration!**

- In the case of a total failure of a wallet server:
  - Restore the failed wallet(s) from seed
  - Restart the vaterstakepool process to allow automatic syncing to occur.

## IRC

- irc.freenode.net
- channel #vatercoin

## Issue Tracker

The [integrated github issue tracker](https://github.com/vatercoin/vaterstakepool/issues)
is used for this project.

## License

vaterstakepool is licensed under the [copyfree](http://copyfree.org) MIT/X11 and
ISC Licenses.

## Version History

- 1.1.0 - Architecture change.
  * Per-ticket votebits were removed in favor of per-user voting preferences.
    A voting page was added and the API upgraded to v2 to support getting and
    setting user voting preferences.
  * Addresses from the wallet servers which are needed for generating the 1-of-2
    multisig ticket address are now derived from the new votingwalletextpub
    config option. This removes the need to call getnewaddress on each wallet.
  * An experimental daemon (stakepoold) that votes according to user preference
    is available for testing on testnet. This daemon is not for use on mainnet
    at this time.
- 1.0.0 - Major changes/improvements.
  * API is now at v1 status.  API Tokens are generated for all users with a
    verified email address when upgrading.  Tokens are generated for new
    users on demand when visiting the Settings page which displays their token.
    Authenticated users may use the API to submit a public key address and to
    retrieve ticket purchasing information.  The voting service's stats are also
    available through the API without authentication.
- 0.0.4 - Major changes/improvements.
  * config.toml is no longer required as the options in that file have been
    migrated to vaterstakepool.conf.
  * Automatic syncing of scripts, tickets, and vote bits is now performed at
    startup.  Syncing of vote bits is a long process and can be skipped with the
    SkipVoteBitsSync flag/configuration value.
  * Temporary wallet connectivity errors are now handled much more gracefully.
  * A preliminary v0.1 API was added.
- 0.0.3 - More expected/basic web application functionality added.
  * SMTPHost now defaults to an empty string so a voting service can be used for
    development or testing purposes without a configured mail server.  The
    contents of the emails are sent through the logger so links can still be
    followed.
  * Upon sign up, users now have an email sent with a validation link.
    They will not be able to sign in until they verify.
  * New settings page that allows users to change their email address/password.
  * Bug fix to HeightRegistered migration for users who signed up but never
    submitted an address would not be able to login.
- 0.0.2 - Minor improvements/feature addition
  * The importscript RPC is now called with the current block height at the
    time of user registration. Previously, importscript triggered a rescan
    for transactions from the genesis block.  Since the user just registered,
    there won't be any transactions present.  A new HeightRegistered column
    is automatically added to the Users table.  A default value of 15346 is
    used for existing users who already had a multisigscript generated.
    This can be adjusted to a more reasonable value for you pool by running
    the following MySQL query:
    ```UPDATE Users SET HeightRegistered = NEWVALUE WHERE HeightRegistered = 15346;```
  * Users may now reset their password by specifying an email address and
    clicking a link that they will receive via email.  You will need to
    add a proper configuration for your mail server for it to work properly.
    The various SMTP options can be seen in **sample-vaterstakepool.conf**.
  * User instructions on the address and ticket pages were updated.
  * SpentBy link added to the voted tickets display.
- 0.0.1 - Initial release for mainnet operations
