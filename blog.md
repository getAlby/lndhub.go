# LNDhub.go
Today we are announcing lndhub.go, a Go implementation of an LNDHub server.

First [released](https://bluewallet.io/BlueWallet-brings-zero-configuration-Lightning-payments-to-iOS-and-Android-30137a69f071/) in 2018, Bluewallet was one of the first wallets to implement support for the Lightning Network on both Android and iOS. The Bluewallet team pragmatically asserted at the time that for amounts in the context of day-to-day spending, the UX improvement of a custodial Lightning wallet far outweighs the risks that come with it. Open-sourcing their back-end meant that anyone could deploy their own instance and connect Bluewallet directly to their own wallet, or even start their own lightning-bank for their family, friends or community.

At Alby we leveraged the simplicity of the LNDHub API to allow users to import their existing Bluewallet Lightning wallet directly into their browser and experience the Lightning-native web. Vice versa, users that don't yet have a Lightning wallet can create an Alby-hosted wallet and import it into the Bluewallet or Zeus mobile applications. This way, your Lightning wallet is always with you, no matter what platform you are on. 

We have decided to write an implementation of LNDHub in Go that focusses on simplicity, maintainability and ease of deployment.
Their are multiple reasons why this seems like a good idea:

* No runtime dependencies (all you need is a single binary executable), as opposed to the current implementation which requires a NodeJS runtime.
* Use of an ORM ([bun](https://bun.uptrace.dev/)) to support deployments with SQLite and PostgreSQL as databases, a more conventional approach than using Redis.
* Plan for multiple node backends, where LNDHub currently only supports LND.
* Using constraints and functions in the DB to prevent inconsistent data.
* Extensibility to add more features or plugins later on.
* The Bluewallet team has currenlty shifted it's attention more towards a non-custodial mobile wallet using LDK.