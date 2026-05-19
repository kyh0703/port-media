package exception

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e AppError) Error() string {
	return e.Message
}

func New(code string, message string, status int) AppError {
	return AppError{Code: code, Message: message, Status: status}
}
