package rpcpattern

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"testing"
	"time"
)

func TestDeadlineAndCorrelation(t *testing.T) {
	ctx, cancel, id := RequestContext(context.Background(), time.Millisecond)
	defer cancel()
	md, _ := metadata.FromOutgoingContext(ctx)
	if len(id) != 32 || md.Get("x-request-id")[0] != id {
		t.Fatal("correlation missing")
	}
	<-ctx.Done()
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatal(ctx.Err())
	}
}
func TestTransportErrorMapping(t *testing.T) {
	cases := map[codes.Code]int{codes.FailedPrecondition: http.StatusConflict, codes.Unavailable: http.StatusServiceUnavailable, codes.DeadlineExceeded: http.StatusGatewayTimeout, codes.InvalidArgument: http.StatusBadRequest}
	for code, want := range cases {
		if got := HTTPStatus(status.Error(code, "private provider detail")); got != want {
			t.Fatal(code, got)
		}
	}
}
