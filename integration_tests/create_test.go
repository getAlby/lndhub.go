package integration_tests

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/migrate"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
)

func TestCreateAndAuthUser(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	ctxEcho := e.NewContext(req, rec)

	lndHubService, err := LndHubServiceInit()
	assert.Nil(t, err)

	createUserCtrl := controllers.NewCreateUserController(lndHubService)
	authCtrl := controllers.NewAuthController(lndHubService)

	t.Run("success create new user", func(t *testing.T) {
		err := createUserCtrl.CreateUser(ctxEcho)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	req = httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(rec.Body.String()))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	ctxEcho = e.NewContext(req, rec)

	t.Run("success authenticate user", func(t *testing.T) {
		err := authCtrl.Auth(ctxEcho)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func LndHubServiceInit() (*service.LndhubService, error) {
	c := &service.Config{}

	err := godotenv.Load("../.env")
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}
	err = envconfig.Process("", c)
	if err != nil {
		return nil, fmt.Errorf("failed to process env: %w", err)
	}

	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init migrations: %w", err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.TestLNDAddress,
		MacaroonHex: c.TestLNDMacaroonHex,
		CertHex:     c.TestLNDCertHex,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lnd service client: %w", err)
	}

	return &service.LndhubService{
		DB:        dbConn,
		LndClient: &lndClient,
	}, nil
}
