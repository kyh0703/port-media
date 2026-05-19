package response

type Body struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	Error      any    `json:"error,omitempty"`
}

func OK(data any) Body {
	return Success(200, "OK", data)
}

func Created(data any) Body {
	return Success(201, "Created", data)
}

func Success(statusCode int, message string, data any) Body {
	return Body{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

func Error(statusCode int, message string, err any) Body {
	return Body{
		StatusCode: statusCode,
		Message:    message,
		Error:      err,
		Data:       nil,
	}
}
