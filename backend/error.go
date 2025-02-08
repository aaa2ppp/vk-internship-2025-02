package main

type httpError struct {
	Status  int
	Message string
}

func (e httpError) Error() string {
	return e.Message
}

var (
	errInternalError = &httpError{500, "internal error"}
	errBadRequest    = &httpError{400, "bad request"}
	errNotFound      = &httpError{404, "not found"}
)
