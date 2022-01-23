# Rebuilding LndHub

Goal of this project is to build a simple accounting system with a [LNDHub](https://github.com/BlueWallet/LndHub) compatible API that focusses on simplicity, maintainability and ease of deployment.

[LNDHub](https://github.com/BlueWallet/LndHub) is a simple accounting system for LND. It allows users to send and receive lightning payments. Through the API people can access funds through a shared lightning node. (see overview.png diagram)

Some design goals:

* No runtime dependencies (all compiled into a single, simple deployable executable)
* Use of an ORM ([bun](https://bun.uptrace.dev/)?)to support deployments with SQLite and PostgreSQL (default) as databases
* Focus on offchain payments (no onchain transactions supported)
* Plan for multiple node backends ([LND](https://github.com/lightningnetwork/lnd/) gRPC interface is the first implementation) (also through Tor)
* Admin panel for better Ops
* All configuration stored in the DB
* Using constraints and functions in the DB to prevent inconsistent data



### API endpoints

See [LNDHub API](https://github.com/BlueWallet/LndHub/blob/master/controllers/api.js) for enpoints and request/response signatures.

#### /create
Create a new user account

#### /auth
Get new "session" access/refresh tokens. access token is required for all other API endpoints

#### /addinvoice
Generate a new lightning invoice

#### /payinvoice
Pay a lightning invoice

#### /checkpayment/:payment_hash
Check the status of an incoming transaction

#### /balance
Get the user's balanc

#### /gettxs
Get all transactions

#### /getuserinvoices
Get all user's invoices (incoming invoices)

#### /getinfo
Node information



### Datamodel

* Double entry accounting?
	+ https://gist.github.com/NYKevin/9433376
	+ https://gocardless.com/guides/posts/double-entry-bookkeeping/
	+
* see `db/migrations/*`


### Links

* [LNDHub](https://github.com/BlueWallet/LndHub) - Current nodejs implementation
