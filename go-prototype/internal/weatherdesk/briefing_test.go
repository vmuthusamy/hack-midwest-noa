package weatherdesk

import (
	"bytes"
	"context"
	"encoding/json"
	pb "github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/gen/weatherdeskv1"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type qaBriefingService struct {
	pb.UnimplementedEvidenceBriefingServiceServer
	explain func(context.Context, *pb.ExplainRequest) (*pb.ExplainResponse, error)
}

func (s *qaBriefingService) GetStatus(context.Context, *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Configured: true, Installed: true}, nil
}
func (s *qaBriefingService) Explain(ctx context.Context, r *pb.ExplainRequest) (*pb.ExplainResponse, error) {
	return s.explain(ctx, r)
}
func qaBriefingResponse(r *pb.ExplainRequest) *pb.ExplainResponse {
	return &pb.ExplainResponse{RequestId: r.RequestId, FacilityId: r.Facility.Id, EvidenceTimestamp: r.EvidenceTimestamp, ReviewRevision: r.ReviewRevision, Briefing: &pb.Briefing{Explanation: "Transport fixture; not model output."}}
}
func qaBriefingApp(t *testing.T, explain func(context.Context, *pb.ExplainRequest) (*pb.ExplainResponse, error)) *App {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterEvidenceBriefingServiceServer(server, &qaBriefingService{explain: explain})
	go server.Serve(listener)
	app, err := New(t.TempDir(), listener.Addr().String())
	if err != nil {
		server.Stop()
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close(); server.Stop(); listener.Close() })
	return app
}
func qaJSONRequest(app *App, path string, body map[string]string) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	return w
}
func TestBriefingLiveSnapshotIdentityAndNoReplayContext(t *testing.T) {
	captured := make(chan *pb.ExplainRequest, 1)
	app := qaBriefingApp(t, func(_ context.Context, r *pb.ExplainRequest) (*pb.ExplainResponse, error) {
		captured <- r
		return qaBriefingResponse(r), nil
	})
	f := app.replays["barnwell"].Facility
	refresh := time.Now().UTC().Format(time.RFC3339)
	app.live["barnwell"] = Evidence{Mode: "LIVE", Facility: f, LastRefresh: refresh, Alerts: []Alert{}, Gaps: []string{"No prior approval connected."}}
	w := qaJSONRequest(app, "/api/briefing?facility=barnwell", map[string]string{"mode": "LIVE"})
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	r := <-captured
	if r.Mode != "LIVE" || r.Facility.Id != f.ID || r.EvidenceTimestamp != refresh || r.PriorApproval != nil || r.ReviewPolicy != nil || len(r.Recipients) != 0 {
		t.Fatal("LIVE request mixed context", r)
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["facilityId"] != f.ID || result["evidenceTimestamp"] != refresh || result["mode"] != "LIVE" {
		t.Fatal("LIVE response lost identity", result)
	}
}
func TestBriefingReplayCarriesExactContextWithoutMutation(t *testing.T) {
	captured := make(chan *pb.ExplainRequest, 1)
	app := qaBriefingApp(t, func(_ context.Context, r *pb.ExplainRequest) (*pb.ExplainResponse, error) {
		captured <- r
		return qaBriefingResponse(r), nil
	})
	replay := app.replays["barnwell"]
	revision := digest(packageFor(replay))
	before := digest(replay)
	w := qaJSONRequest(app, "/api/briefing?facility=barnwell", map[string]string{"mode": "REPLAY", "revision": revision})
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	r := <-captured
	if r.Mode != "REPLAY" || r.PriorApproval == nil || r.PriorApproval.Instruction != "Report at 10:00" || len(r.PriorApproval.RecipientIds) != 5 || len(r.Recipients) != 6 || r.ReviewPolicy == nil || !r.ReviewPolicy.Eligible || r.ReviewRevision != revision || r.EvidenceTimestamp != replay.CapturedAt {
		t.Fatal("Incomplete replay review context", r)
	}
	if len(r.EvidenceGaps) == 0 || !strings.Contains(r.EvidenceGaps[0], "T+0 approval, T+5 warning") || !strings.Contains(r.EvidenceGaps[0], "History timestamps record local interactions") {
		t.Fatal("Replay chronology not explicit", r.EvidenceGaps)
	}
	if digest(app.replays["barnwell"]) != before {
		t.Fatal("Generation mutated approval, drafts, notice states or history")
	}
}
func TestBriefingStaleReviewRejectedBeforeWorker(t *testing.T) {
	var calls atomic.Int32
	app := qaBriefingApp(t, func(_ context.Context, r *pb.ExplainRequest) (*pb.ExplainResponse, error) {
		calls.Add(1)
		return qaBriefingResponse(r), nil
	})
	w := qaJSONRequest(app, "/api/briefing?facility=barnwell", map[string]string{"mode": "REPLAY", "revision": "stale-review"})
	if w.Code != http.StatusConflict || calls.Load() != 0 {
		t.Fatal(w.Code, calls.Load())
	}
}
func TestBriefingMaterialChangeDuringGenerationRejectsOutput(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	app := qaBriefingApp(t, func(ctx context.Context, r *pb.ExplainRequest) (*pb.ExplainResponse, error) {
		close(started)
		select {
		case <-release:
			return qaBriefingResponse(r), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	revision := digest(packageFor(app.replays["barnwell"]))
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- qaJSONRequest(app, "/api/briefing?facility=barnwell", map[string]string{"mode": "REPLAY", "revision": revision})
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("Worker did not receive request")
	}
	changed := qaJSONRequest(app, "/api/replay/action?facility=barnwell", map[string]string{"action": "case", "case": "update", "revision": revision})
	close(release)
	if changed.Code != http.StatusOK {
		t.Fatal(changed.Code, changed.Body.String())
	}
	select {
	case w := <-done:
		if w.Code != http.StatusConflict {
			t.Fatal("Stale generated output accepted", w.Code, w.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Generation did not complete")
	}
	if app.replays["barnwell"].Approval != nil || len(app.replays["barnwell"].Intents) != 0 {
		t.Fatal("Generation authorized an operation")
	}
}
