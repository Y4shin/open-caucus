package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
)

func ErrorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return nil, apierrors.ToConnect(err)
			}
			return resp, nil
		}
	}
}
