package responses

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Error   bool   `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ErrorResponseLnurl struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

var LnurlpBadArgumentsError = ErrorResponseLnurl{
	Status: "ERROR",
	Reason: "Bad arguments",
}

var LnurlpBadAuthError = ErrorResponseLnurl{
	Status: "ERROR",
	Reason: "Bad auth",
}

var GeneralServerError = ErrorResponse{
	Error:   true,
	Code:    6,
	Message: "Something went wrong. Please try again later",
}

var BadArgumentsError = ErrorResponse{
	Error:   true,
	Code:    8,
	Message: "Bad arguments",
}

var BadAuthError = ErrorResponse{
	Error:   true,
	Code:    1,
	Message: "bad auth",
}

var NotEnoughBalanceError = ErrorResponse{
	Error:   true,
	Code:    2,
	Message: "not enough balance. Make sure you have at least 1% reserved for potential fees",
}

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	c.Logger().Error(err)
	if hub := sentryecho.GetHubFromContext(c); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("UserID", c.Get("UserID"))
			hub.CaptureException(err)
		})
	}
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		c.JSON(code, he.Message)
	} else {
		c.JSON(http.StatusInternalServerError, GeneralServerError)
	}
	// TODO: use an error matching the error code
}
