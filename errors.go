package workers

import (
	"errors"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrClosed          = errors.New("server is closed")
)
