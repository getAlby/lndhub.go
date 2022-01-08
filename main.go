package main

import (
	"os"

	"github.com/bumi/lndhub.go/database"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/middlewares"
	"github.com/bumi/lndhub.go/routes"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

func init() {
	godotenv.Load(".env")
}

func main() {
	db, err := database.Connect(os.Getenv("DATABASE_URI"))
	if err != nil {
		logrus.Errorf("failed to connect with database: %v", err)
		return
	}

	e := echo.New()

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middlewares.ContextDB(db))
	e.Use(middleware.BodyLimit("250K"))

	routes.Routes(e.Group(""))

	e.Logger.Fatal(e.Start(":3000"))
}
