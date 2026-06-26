package errors

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindTransient Kind = "transient"
	KindPermanent Kind = "permanent"
	KindNotFound  Kind = "not_found"
	KindQuota     Kind = "quota"
)

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func IsKind(err error, kind Kind) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind == kind
}

func IsTransient(err error) bool {
	return IsKind(err, KindTransient)
}
