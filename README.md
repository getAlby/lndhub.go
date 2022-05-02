<div align="center">
   <img alt="LNDHub.go" src="static/img/logo.png" width="400">
</div>

# Wrapper for Lightning Network Daemon (lnd) ⚡  

It provides separate accounts with minimum trust for end users.
Live deployment at [ln.getalby.com](https://ln.getalby.com).

### [LndHub](https://github.com/BlueWallet/LndHub) compatible API implemented in Go using relational database backends

* Using a relational database (PostgreSQL and SQLite)
* Focussing only on Lightning (no onchain functionality)
* No runtime dependencies (simple Go executable)
* Extensible to add more features 

### Status: alpha 

## Known Issues

* Fee reserves are not checked prior to making the payment. This can cause a user's balance to go below 0.

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
+ `ENABLE_PROMETHEUS`: (default: false) Enable Prometheus metrics to be exposed
+ `PROMETHEUS_PORT`: (default: 9092) Prometheus port (path: `/metrics`)
+ `WEBHOOK_URL`: Optional. Callback URL for incoming and outgoing payment events, see below.
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

## Prometheus

Prometheus metrics can be optionally exposed through the `ENABLE_PROMETHEUS` environment variable.
For an example dashboard, see https://grafana.com/grafana/dashboards/10913.

## Webhooks

If `WEBHOOK_URL` is specified, a http POST request will be dispatched at that location when an incoming payment is settled, or an outgoing payment is completed. Example payload:

```
{
  "id": 690,
  "type": "outgoing",
  "user_id": 286,
  "User": null,
  "amount": 10,
  "fee": 0,
  "memo": "memo",
  "description_hash": "",
  "payment_request": "lnbcrt100n1p3xjtw3pp5s56uj7d4lpchz0d25jggq0au5r65skvtp8f5dvm5e3l4c6cxg9aqdqy09eqcqzpgxqrrsssp53pyhm6j2vl4sr7ul8pa4sfvptk96yn2lkeceh8z2etkl8emapkgq9qyyssqgpxy39ktu60gkct8y7ehkemu5c77dffg905cd6sr5cukgjna2nwnfew65zkems5sm5xmdllrkf8ym3dhc2asj7hn27tq7xe0dq5hq5spt2g4c2",
  "destination_pubkey_hex": "025c1d5d1b4c983cc6350fc2d756fbb59b4dc365e45e87f8e3afe07e24013e8220",
  "DestinationCustomRecords": null,
  "r_hash": "8535c979b5f871713daaa490803fbca0f548598b09d346b374cc7f5c6b06417a",
  "preimage": "fa40fd77183b1bae11fec8a1479f08e210f1711f07a5a6a45b8d46a34cd820b1",
  "internal": false,
  "keysend": false,
  "state": "settled",
  "error_message": "",
  "add_index": 0,
  "CreatedAt": "2022-04-27T13:50:43.938597+02:00",
  "ExpiresAt": "2022-04-27T14:49:37+02:00",
  "updated_at": "2022-04-27T13:50:44.313549+02:00",
  "settled_at": "2022-04-27T13:50:44.313539+02:00"
}
```

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

