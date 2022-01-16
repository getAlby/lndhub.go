package integration_tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/bumi/lndhub.go/controllers"
	"github.com/bumi/lndhub.go/db"
	"github.com/bumi/lndhub.go/lib"
)

func TestCreateUser(t *testing.T) {
	e := echo.New()

	err := godotenv.Load("../.env")
	if err != nil {
		logrus.Fatal("failed to get env value")
	}
	dbConn, err := db.Open(fmt.Sprintf("../%s", os.Getenv("DATABASE_URI")))
	if err != nil {
		logrus.Fatalf("failed to connect to database: %v", err)
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	ctx := &lib.IndhubContext{
		Context: c,
		DB:      dbConn,
	}

	c.SetParamNames("login", "password")
	c.SetParamValues("test-login", "test-password")

	if assert.NoError(t, controllers.CreateUserController{}.CreateUser(ctx)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
