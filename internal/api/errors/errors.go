package apierrors

import (
	"errors"
	"net/http"

	connect "connectrpc.com/connect"
)

type Kind string

const (
	KindInternal         Kind = "internal"
	KindInvalidArgument  Kind = "invalid_argument"
	KindUnauthenticated  Kind = "unauthenticated"
	KindPermissionDenied Kind = "permission_denied"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindUnimplemented    Kind = "unimplemented"
)

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func classify(err error) Kind {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Kind
	}
	return KindInternal
}

func HTTPStatus(err error) int {
	switch classify(err) {
	case KindInvalidArgument:
		return http.StatusBadRequest
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindPermissionDenied:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnimplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func ConnectCode(err error) connect.Code {
	switch classify(err) {
	case KindInvalidArgument:
		return connect.CodeInvalidArgument
	case KindUnauthenticated:
		return connect.CodeUnauthenticated
	case KindPermissionDenied:
		return connect.CodePermissionDenied
	case KindNotFound:
		return connect.CodeNotFound
	case KindConflict:
		return connect.CodeFailedPrecondition
	case KindUnimplemented:
		return connect.CodeUnimplemented
	default:
		return connect.CodeInternal
	}
}

func ToConnect(err error) error {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(ConnectCode(err), err)
}
