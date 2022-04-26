module github.com/getAlby/lndhub.go

go 1.17

// +heroku goVersion go1.17

require (
	github.com/getsentry/sentry-go v0.12.0
	github.com/go-playground/validator/v10 v10.10.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/joho/godotenv v1.4.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/labstack/echo/v4 v4.7.2
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
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	google.golang.org/grpc v1.43.0
	gopkg.in/macaroon.v2 v2.1.0
)

require (
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/SporkHubr/echo-http-cache v0.0.0-20200706100054-1d7ae9f38029
	github.com/go-openapi/spec v0.20.5 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/gorilla/websocket v1.5.0
	github.com/labstack/echo-contrib v0.12.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/swaggo/echo-swagger v1.3.0
	github.com/swaggo/swag v1.8.1
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4 // indirect
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	golang.org/x/tools v0.1.10 // indirect
	google.golang.org/genproto v0.0.0-20220114231437-d2e6a121cae0 // indirect
)
