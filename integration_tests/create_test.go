package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/migrate"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
)

func TestCreateUser(t *testing.T) {
	c := &service.Config{}
	e := echo.New()

	// Load configruation from environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Errorf("Failed to load .env file")
	}
	err = envconfig.Process("", c)
	if err != nil {
		log.Errorf("Failed to process env")
	}

	dbConn, err := db.Open(c.TestDatabaseUri)
	if err != nil {
		logrus.Fatalf("failed to connect to database: %v", err)
		return
	}
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		panic(err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		panic(err)
	}

	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.TestLNDAddress,
		MacaroonHex: c.TestLNDMacaroonHex,
		CertHex:     c.TestLNDCertHex,
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	ctxEcho := e.NewContext(req, rec)

	createUserService := &service.LndhubService{
		DB:        dbConn,
		LndClient: &lndClient,
	}

	createUserCtrl := controllers.NewCreateUserController(createUserService)

	if assert.NoError(t, createUserCtrl.CreateUser(ctxEcho)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
