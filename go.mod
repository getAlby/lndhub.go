module github.com/getAlby/lndhub.go

go 1.17

// +heroku goVersion go1.17

require (
	github.com/getsentry/sentry-go v0.12.0
	github.com/go-playground/validator/v10 v10.10.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/joho/godotenv v1.4.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.1
	github.com/lightningnetwork/lnd v0.14.1-beta
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v1.0.21
	github.com/uptrace/bun/dialect/pgdialect v1.0.21
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.21
	github.com/uptrace/bun/driver/pgdriver v1.0.21
	github.com/uptrace/bun/driver/sqliteshim v1.0.21
	github.com/uptrace/bun/extra/bundebug v1.0.21
	github.com/ziflex/lecho/v3 v3.1.0
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3
	google.golang.org/grpc v1.43.0
	gopkg.in/macaroon.v2 v2.1.0
)

require (
	github.com/SporkHubr/echo-http-cache v0.0.0-20200706100054-1d7ae9f38029
	github.com/gorilla/websocket v1.5.0
	github.com/fiatjaf/lightningd-gjson-rpc v1.4.1
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/tidwall/gjson v1.6.0
	golang.org/x/net v0.0.0-20220114011407-0dd24b26b47d // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/genproto v0.0.0-20220114231437-d2e6a121cae0 // indirect
)
