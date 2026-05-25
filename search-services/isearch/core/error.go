package core

import "errors"

var (
	ErrInternal      = errors.New("internal error")
	ErrBadRequest    = errors.New("bad request")
	ErrIndexNotReady = errors.New("search index is not ready yet")
	ErrUnavailable   = errors.New("error at server side")
)
