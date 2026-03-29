package myerrors

import "errors"

var (
	ErrNoRows error = errors.New("no rows")
	ErrUserExists error = errors.New("user already exists")
)
