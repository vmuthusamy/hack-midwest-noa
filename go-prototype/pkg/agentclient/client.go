// Package agentclient is the typed, reusable Go client for the Strands worker.
package agentclient

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	pb "weatherdesk.local/prototype/pkg/gen/weatherdeskv1"
	"weatherdesk.local/prototype/pkg/rpcpattern"
)

type Client struct {
	conn    *grpc.ClientConn
	service pb.EvidenceBriefingServiceClient
}

func New(address string) (*Client, error) {
	// Plaintext is restricted to the local loopback worker. Use TLS/mTLS before remote deployment.
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2<<20), grpc.MaxCallSendMsgSize(2<<20)))
	if err != nil {
		return nil, err
	}
	return &Client{conn, pb.NewEvidenceBriefingServiceClient(conn)}, nil
}
func (c *Client) Close() error { return c.conn.Close() }
func (c *Client) Status(ctx context.Context) (*pb.StatusResponse, error) {
	deadline, cancel, _ := rpcpattern.RequestContext(ctx, 3*time.Second)
	defer cancel()
	return c.service.GetStatus(deadline, &pb.StatusRequest{})
}
func (c *Client) Explain(ctx context.Context, input *pb.ExplainRequest) (*pb.ExplainResponse, error) {
	deadline, cancel, id := rpcpattern.RequestContext(ctx, 90*time.Second)
	defer cancel()
	input.RequestId = id
	response, err := c.service.Explain(deadline, input)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Briefing == nil || response.RequestId != id || response.FacilityId != input.Facility.Id || response.EvidenceTimestamp != input.EvidenceTimestamp {
		return nil, status.Error(codes.DataLoss, "Evidence response identity mismatch")
	}
	return response, nil
}
func OutputJSON(output *pb.Briefing) ([]byte, error) {
	return (protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}).Marshal(output)
}
