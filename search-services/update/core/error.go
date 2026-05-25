package core

import "errors"

var (
	ErrInternal      = errors.New("internal error")
	ErrBadRequest    = errors.New("bad request")
	ErrUpdateRunning = errors.New("update already running")
)
