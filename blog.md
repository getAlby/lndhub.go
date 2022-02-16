# LNDhub.go
Today we are announcing lndhub.go, a Go implementation of a LNDhub server.

First [released](https://bluewallet.io/BlueWallet-brings-zero-configuration-Lightning-payments-to-iOS-and-Android-30137a69f071/) in 2018, Bluewallet was one of the first wallets to implement support for the Lightning Network on both Android and iOS. The Bluewallet team pragmatically asserted at the time that for amounts in the context of day-to-day spending, the UX improvement of a custodial Lightning wallet far outweighs the risks that come with it. Open-sourcing their backend meant that anyone could deploy their own instance and connect Bluewallet directly to their own wallet, or even start their own lightning-bank for their family, friends or community.

At Alby we leveraged the simplicity of the LNDhub API to allow users to import their existing Bluewallet Lightning wallets directly into their browser and experience the Lightning-native web. Vice versa, users that don't yet have a Lightning wallet can create an Alby-hosted wallet and import it into the Bluewallet or Zeus mobile applications. This way, your Lightning wallet is always with you, no matter what platform you are on. 

Accounting tools can not only be helpful for external end-users. It can also be used internally in a system architecture, to allow existing applications to send and receive lightning transactions without the need to host your own node. Even if you do host your own node, it might be useful to do interact with it through an accounting layer. LNDhub with it's simple HTTP REST API can be very helpful here.

In general, we believe that community banking concepts and shared-custody of funds among community or circle of friends are very helpful for Bitcoin adoption. LNDhub provides a simple accounting solution that makes it possible for everyone to provide lightning accounts to their friends and family. 

We have decided to write an implementation of LNDhub in Go that focusses on simplicity, maintainability and ease of deployment.
There are multiple reasons why this seems like a good idea:

* No runtime dependencies (all you need is a single binary executable), as opposed to the current implementation which requires a NodeJS runtime.
* Use of an ORM ([bun](https://bun.uptrace.dev/)) to support deployments with SQLite and PostgreSQL as databases, a more conventional approach than using Redis.
* Support multiple node backends, where LNDhub currently only supports LND. (Hello BOLT12 👀)
* Using constraints and functions in the DB to prevent inconsistent data.
* Extensibility to add more features or plugins later on. Why not let the user have fiat-denominated accounts?
* The Bluewallet team has currenlty shifted it's attention more towards a non-custodial mobile wallet using [LDK](https://lightningdevkit.org/).

We have released version 0.1.1 - the "[Bedford FC](https://www.realbedford.com/) fan edition" on GitHub where you can also [follow the development](https://github.com/getAlby/lndhub.go).

If you have feedback how a simple accounting system can be helpful for you or if you have feature ideas, then please [let us know](https://github.com/getAlby/lndhub.go/issues).
