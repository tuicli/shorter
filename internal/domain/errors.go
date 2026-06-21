package domain

import "errors"

var (
	ErrTooManyLines       = errors.New("too many input lines")
	ErrCodeCollisionLimit = errors.New("code generation retry limit exhausted")
	ErrCodeExists         = errors.New("short code already exists")
	ErrOriginalURLExists  = errors.New("original URL already exists")
	ErrLinkNotFound       = errors.New("short link not found")
	ErrInvalidStatus      = errors.New("invalid short link status")
)
