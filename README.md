# LndHub.go
Wrapper for Lightning Network Daemon (lnd). It provides separate accounts with minimum trust for end users.

Live deployment at [ln.getalby.com](https://ln.getalby.com).

### [LndHub](https://github.com/BlueWallet/LndHub) compatible API implemented in Go using relational database backends

* Using a relational database (PostgreSQL and SQLite)
* Focussing only on Lightning (no onchain functionality)
* No runtime dependencies (simple Go executable)
* Extensible to add more features 

### Status: alpha 

## Known Issues

* Currently no fee handling (users are currently not charged for lightning transaction fees)

## Configuration

All required configuration is done with environment variables and a `.env` file can be used.
Check the `.env_example` for an example.

```shell
cp .env_example .env
vim .env # edit your config
```

### Available configuration

+ `DATABASE_URI`: The URI for the database. If you want to use SQLite use for example: `file:data.db`
+ `JWT_SECRET`: We use [JWT](https://jwt.io/) for access tokens. Configure your secret here
+ `JWT_ACCESS_EXPIRY`: How long the access tokens should be valid (in seconds, default 2 days)
+ `JWT_REFRESH_EXPIRY`: How long the refresh tokens should be valid (in seconds, default 7 days)
+ `LND_ADDRESS`: LND gRPC address (with port) (e.g. localhost:10009)
+ `LND_MACAROON_HEX`: LND macaroon (hex)
+ `LND_CERT_HEX`: LND certificate (hex)
+ `CUSTOM_NAME`: Name used to overwrite the node alias in the getInfo call
+ `LOG_FILE_PATH`: (optional) By default all logs are written to STDOUT. If you want to log to a file provide the log file path here
+ `SENTRY_DSN`: (optional) Sentry DSN for exception tracking
+ `PORT`: (default: 3000) Port the app should listen on
+ `DEFAULT_RATE_LIMIT`: (default: 10) Requests per second rate limit
+ `STRICT_RATE_LIMIT`: (default: 10) Requests per burst rate limit (e.g. 1 request each 10 seconds)
+ `BURST_RATE_LIMIT`: (default: 1) Rate limit burst
## Developing

```shell
go run main.go
```

### Building

To build an `lndhub` executable, run the following commands:

```shell
make
```

### Development LND setup

To run your own local lightning network and LND you can use [Lightning Polar](https://lightningpolar.com/) which helps you to spin up local LND instances. 

Alternatively you can also use the [Alby simnetwork](https://github.com/getAlby/lightning-browser-extension/wiki/Test-setup)


## Database
LndHub.go supports PostgreSQL and SQLite as database backend. But SQLite does not support the same data consistency checks as PostgreSQL.

### Ideas
+ Using low level database constraints to prevent data inconsistencies
+ Follow double-entry bookkeeping ideas (Every transaction is a debit of one account and a credit to another one)
+ Support multiple database backends (PostgreSQL for production, SQLite for development and personal/friend setups)

### Data model

```
                                                     ┌─────────────┐                            
                                                     │    User     │                            
                                                     └─────────────┘                            
                                                            │                                   
                                  ┌─────────────────┬───────┴─────────┬─────────────────┐       
                                  ▼                 ▼                 ▼                 ▼       
       Accounts:          ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                          │   Incoming   │  │   Current    │  │   Outgoing   │  │     Fees     │
       Every user has     └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
       four accounts                                                                            
                                                                                                
                           Every Transaction Entry is associated to one debit account and one   
                                                    credit account                             
                                                                                                
                                                 ┌────────────────────────┐                     
                                                 │Transaction Entry       │                     
                                                 │                        │                     
                                                 │+ user_id               │                     
                   ┌────────────┐                │+ invoice_id            │                     
                   │  Invoice   │────────────────▶+ debit_account_id      │                     
                   └────────────┘                │+ credit_account_id     │                     
                                                 │+ amount                │                     
                  Invoice holds the              │+ ...                   │                     
                  lightning related              │                        │                     
                  data                           └────────────────────────┘                     
                                                                                                
```

