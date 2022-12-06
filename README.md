# LndHub.go

![LndHub.go](static/img/logo.png)

Wrapper for Lightning Network Daemon (lnd) ⚡

It provides separate accounts with minimum trust for end users.
Live deployment at [ln.getalby.com](https://ln.getalby.com).

### [LndHub](https://github.com/BlueWallet/LndHub) compatible API implemented in Go using relational database backends

* Using a relational database (PostgreSQL)
* Focussing only on Lightning (no onchain functionality)
* No runtime dependencies (simple Go executable)
* Extensible to add more features

### Status: alpha

## Configuration

All required configuration is done with environment variables and a `.env` file can be used.
Check the `.env_example` for an example.

```shell
cp .env_example .env
vim .env # edit your config
```

### Available configuration

+ `DATABASE_URI`: The URI for the database. (eg. `postgresql://user:password@localhost:5432/lndhub?sslmode=disable`)
+ `JWT_SECRET`: We use [JWT](https://jwt.io/) for access tokens. Configure your secret here
+ `JWT_ACCESS_EXPIRY`: How long the access tokens should be valid (in seconds, default 2 days)
+ `JWT_REFRESH_EXPIRY`: How long the refresh tokens should be valid (in seconds, default 7 days)
+ `LND_ADDRESS`: LND gRPC address (with port) (e.g. `localhost:10009`)
+ `LND_MACAROON_HEX`: LND macaroon (hex-encoded contents of `admin.macaroon` or `lndhub.macaroon`, see below)
+ `LND_MACAROON_FILE`: LND macaroon (provided as path on a filesystem)
+ `LND_CERT_HEX`: LND certificate (hex-encoded contents of `tls.cert`)
+ `LND_CERT_FILE`: LND certificate (provided as path on a filesystem)
+ `CUSTOM_NAME`: Name used to overwrite the node alias in the getInfo call
+ `LOG_FILE_PATH`: (optional) By default all logs are written to STDOUT. If you want to log to a file provide the log file path here
+ `SENTRY_DSN`: (optional) Sentry DSN for exception tracking
+ `HOST`: (default: "localhost:3000") Host the app should listen on
+ `PORT`: (default: 3000) Port the app should listen on
+ `DEFAULT_RATE_LIMIT`: (default: 10) Requests per second rate limit
+ `STRICT_RATE_LIMIT`: (default: 10) Requests per burst rate limit (e.g. 1 request each 10 seconds)
+ `BURST_RATE_LIMIT`: (default: 1) Rate limit burst
+ `ENABLE_PROMETHEUS`: (default: false) Enable Prometheus metrics to be exposed
+ `PROMETHEUS_PORT`: (default: 9092) Prometheus port (path: `/metrics`)
+ `WEBHOOK_URL`: Optional. Callback URL for incoming and outgoing payment events, see below.
+ `FEE_RESERVE`: (default: false) Keep fee reserve for each user
+ `ALLOW_ACCOUNT_CREATION`: (default: true) Enable creation of new accounts
+ `ADMIN_TOKEN`: Only allow account creation requests if they have the header `Authorization: Bearer ADMIN_TOKEN`
+ `MIN_PASSWORD_ENTROPY`: (default: 0 = disable check) Minimum entropy (bits) of a password to be accepted during account creation
+ `MAX_RECEIVE_AMOUNT`: (default: 0 = no limit) Set maximum amount (in satoshi) for which an invoice can be created
+ `MAX_SEND_AMOUNT`: (default: 0 = no limit) Set maximum amount (in satoshi) of an invoice that can be paid
+ `MAX_ACCOUNT_BALANCE`: (default: 0 = no limit) Set maximum balance (in satoshi) for each account

### Macaroon

There are two ways how to obtain hex-encoded macaroon needed for `LND_MACAROON_HEX`.

Either you hex-encode the `admin.macaroon`:

```
xxd -p -c 1000 ~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon
```

Or you bake a new macaroon with the following permissions and use that instead:

```
lncli bakemacaroon info:read invoices:read invoices:write offchain:read offchain:write
```

If you want to use a macaroon stored on a filesystem, you either set `LND_MACAROON_FILE` to a path pointing to `admin.macaroon`
or use the following command to generate the `lndhub.macaroon` and setting the variable to path of that file.

```
lncli bakemacaroon --save_to=lndhub.macaroon info:read invoices:read invoices:write offchain:read offchain:write
```

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

LndHub.go requires a PostgreSQL database backend.

## Prometheus

Prometheus metrics can be optionally exposed through the `ENABLE_PROMETHEUS` environment variable.
For an example dashboard, see https://grafana.com/grafana/dashboards/10913.

## Webhooks

If `WEBHOOK_URL` is specified, a http POST request will be dispatched at that location when an incoming payment is settled, or an outgoing payment is completed. Example payload:

```
{
  "id": 721,
  "type": "incoming", //incoming, outgoing
  "user_id": 299,
  "amount": 1000,
  "fee": 0,
  "memo": "fill wallet",
  "description_hash": "",
  "payment_request": "lnbcrt10u1p38p4ehpp5xp07pda02vk40wxd9gyrene8qzheucz7ast435u9jwxejs6f0v5sdqjve5kcmpqwaskcmr9wscqzpgxqyz5vqsp56nyve3v5fw306j74nmewv7t5ey3aer2khjrrwznh4k2vuw44unzq9qyyssqv2wq9hn7a39x8cvz9fvpzul87u4kc4edf0t6jukzvmx8v5swl3jqg8p3sh6czkepczcjkm523q9x8yswsastctnsns3q9d26szu703gpwh7a09",
  "destination_pubkey_hex": "0376442c750766d5d127512609a5618d9aa82db2d06aae226084da92a3e133acda",
  "custom_records": {
    "5482373484": "YWY4MDhlZDUxZjNmY2YxNWMxYWI3MmM3ODVhNWI1MDE="
  }, //only set when keysend=true
  "r_hash": "305fe0b7af532d57b8cd2a083ccf2700af9e605eec1758d385938d9943497b29",
  "preimage": "3735363531303032626332356439376136643461326434336335626434653035",
  "keysend": false,
  "state": "settled",
  "created_at": "2022-05-03T09:18:15.15774+02:00",
  "expires_at": "2022-05-04T09:18:15.157597+02:00",
  "updated_at": "2022-05-03T09:18:19.837567+02:00",
  "settled_at": "2022-05-03T09:18:19+02:00"
}
```

## Keysend

Both incoming and outgoing keysend payments are supported. For outgoing keysend payments, check out the [API documentation](https://ln.getalby.com/swagger/index.html#/Payment/post_keysend).

For incoming keysend payments, we are using a [custom TLV record with type `696969`](https://github.com/satoshisstream/satoshis.stream/blob/main/TLV_registry.md#field-696969---lnpay), which should contain the hex-encoded `login` of the receiving user's account. TLV records are stored as json blobs with the invoices and are returned by the `/getuserinvoices` endpoint.

The V2 API has an endpoint to make multiple keysend payments with 1 request, which can be useful for splitting value4value payments.

### Ideas

+ Using low level database constraints to prevent data inconsistencies
+ Follow double-entry bookkeeping ideas (Every transaction is a debit of one account and a credit to another one)

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
