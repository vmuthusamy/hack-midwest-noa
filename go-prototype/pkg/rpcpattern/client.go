// Package rpcpattern holds transport patterns shared by service clients.
package rpcpattern

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RequestContext propagates a correlation ID and a bounded deadline. No automatic retries.
func RequestContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, string) {
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	id := hex.EncodeToString(idBytes)
	ctx, cancel := context.WithTimeout(parent, timeout)
	return metadata.AppendToOutgoingContext(ctx, "x-request-id", id), cancel, id
}

func HTTPStatus(err error) int {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.FailedPrecondition:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Canceled:
		return http.StatusRequestTimeout
	default:
		return http.StatusBadGateway
	}
}
