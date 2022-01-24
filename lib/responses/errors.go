package responses

type ErrorResponse struct {
	Error   bool   `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
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
	Message: "not enough balance. Make sure you have at least 1%% reserved for potential fees",
}
