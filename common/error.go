package common

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrAccess    = errors.New("unauthorized")
	ErrExists    = errors.New("already exists")
	ErrNotExists = errors.New("does not exist")
	ErrCanceled  = errors.New("execution canceled")
	ErrInvalid   = errors.New("invalid argument")
	ErrLimit     = errors.New("too many requests")
)

func AuthorizationError(detail string, args ...any) error {
	return Error(ErrAccess, detail, args...)
}

func DuplicateEntryError(detail string, args ...any) error {
	return Error(ErrExists, detail, args...)
}

func DoesNotExistError(detail string, args ...any) error {
	return Error(ErrNotExists, detail, args...)
}

func InvalidArgument(detail string, args ...any) error {
	return Error(ErrInvalid, detail, args...)
}

func TooManyRequests(detail string, args ...any) error {
	return Error(ErrLimit, detail, args...)
}

func IsError(err error, target error) bool {
	if err == nil && target != nil {
		return false
	}

	return errors.Is(err, target)
}

func Error(err error, detail string, args ...any) error {
	if err == nil {
		return nil
	}

	return errors.Wrap(err, fmt.Sprintf(detail, args...))
}

func Must(err error) {
	if err != nil {
		panic(Error(err, "Error must not be raised"))
	}
}

func MustReturn[T any](value T, err error) T {
	if err != nil {
		panic(Error(err, "Error must not be raised"))
	}
	return value
}
