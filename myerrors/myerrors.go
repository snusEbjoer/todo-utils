package myerrors

import "errors"

var ErrUsernameNotUnique = errors.New("err: username not unique")
