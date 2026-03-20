package auth

type UnauthorizedError struct {
	Code    int
	Message string
}

func (e *UnauthorizedError) Error() string {
	return e.Message
}
