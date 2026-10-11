package agentclient

import (
	"context"
	"encoding/json"
	"time"

	pb "github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/gen/weatherdeskv1"
	"github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/rpcpattern"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Metrics reads the worker's allowlisted measurements; it never invokes a model.
func (c *Client) Metrics(ctx context.Context) (json.RawMessage, error) {
	deadline, cancel, _ := rpcpattern.RequestContext(ctx, 5*time.Second)
	defer cancel()
	response, err := c.service.GetMetrics(deadline, &pb.MetricsRequest{})
	if err != nil {
		return nil, err
	}
	if response == nil || response.Metrics == nil {
		return nil, status.Error(codes.DataLoss, "Missing inference measurements")
	}
	return json.Marshal(response.Metrics.AsMap())
}
