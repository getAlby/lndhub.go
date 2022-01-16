# Rebuilding LNDHub

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



### ToDos

- [ ] Project setup for [Echo](https://echo.labstack.com/), [bun](https://bun.uptrace.dev/) (with support for PostgreSQL and SQLite), Unit-Test setup
- [ ] Implement first endpoints (`/create`, `/auth`, `/addinvoice`)
- [ ] Connect to LND (gRPC API) (in the future the API implementation should be configurable)
- [ ] ...


### Datamodel

* Double entry accounting?
	+ https://gist.github.com/NYKevin/9433376
	+ https://gocardless.com/guides/posts/double-entry-bookkeeping/
	+

#### users

```
+ id (primary key)
+ email (optional)
+ login (auto generated, random string)
+ password (auto generated, random string)
+ refresh_token (auto generated on /auth call)
+ access_token (auto generated on /auth call)
+ updated_at (datetime)
+ created_at (datetime)
```

#### tokens

```
+ id (primary key)
+ user_id (foreign_key)
+ name (string - optional name of the application)
+ access_token (string auto generated on /auth call)
+ refresh_token (string auto generated on /auth call)
+ expires_at (datetime / created_at + 2 weeks)
+ created_at (datetime)
```

#### invoices

```
+ id (primary key)
+ type (enum: incoming, outgoing)
+ user_id (foreign key, constaint)
+ transaction_entry_id (foreign key)
+ amount (integer, constraint >=0)
+ memo (string)
+ description_hash (string)
+ payment_request (string)
+ r_hash (string)
+ state (string (enum: )
+ created_at (datetime)
+ expires_at (datetime)
+ settled_at (datetime)
```

#### accounts
```
+ user_id
+ type (enum: outgoing, incoming, current)
```

#### transaction_entries

```
+ user_id
+ invoice_id
+ credit_account_id
+ debit_account_id
+ amount (integer >0 constraint)
+ created_at (datetime)
```


### Know Issues

#### Self-payments
How do we identify self-payments?
We could use two different lightning wallets for sending and receiving. This would avoid the need to identify self-payments completely and any payment would be a proper lightning payment.

### Links

* [LNDHub](https://github.com/BlueWallet/LndHub) - Current nodejs implementation
