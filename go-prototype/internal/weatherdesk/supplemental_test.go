package weatherdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var riverCheckTime = time.Date(2026, 10, 11, 0, 35, 0, 0, time.UTC)

// Representative public-response shapes, not a claim of present conditions.
const barnwellRiverMetadata = `{"lid":"SRBS1","usgsId":"02175200","flood":{"stageUnits":"ft","categories":{"action":{"stage":-9999},"minor":{"stage":-9999},"moderate":{"stage":-9999},"major":{"stage":-9999}}}}`
const floridaRiverMetadata = `{"lid":"STEF1","usgsId":"02324000","latitude":29.786389,"longitude":-83.321667,"status":{"observed":{"primary":12.94,"primaryUnit":"ft","validTime":"2026-10-10T23:00:00Z","floodCategory":"no_flooding"}},"flood":{"stageUnits":"ft","categories":{"action":{"stage":17.5},"minor":{"stage":19.5},"moderate":{"stage":23},"major":{"stage":26}}}}`

func riverUSGSFeature(site, parameter, observed, value, unit string) string {
	return fmt.Sprintf(`{"properties":{"monitoring_location_id":%q,"parameter_code":%q,"time":%q,"value":%s,"unit_of_measure":%q,"approval_status":"Provisional"},"geometry":{"type":"Point","coordinates":[-81.408163,33.240428]}}`, site, parameter, observed, value, unit)
}

func riverFixtureGetter(metadata, observations string) func(context.Context, string, any) error {
	return func(_ context.Context, url string, target any) error {
		body := metadata
		if strings.Contains(url, "waterdata.usgs.gov") {
			body = observations
		}
		return json.Unmarshal([]byte(body), target)
	}
}

func TestBarnwellRiverPreservesMissingThresholdAndRelevance(t *testing.T) {
	observations := `{"features":[` + riverUSGSFeature("USGS-02175200", "00065", "2026-10-10T23:30:00+00:00", `"6.39"`, "ft") + `]}`
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "SC-DEMO-01"}, riverCheckTime, riverFixtureGetter(barnwellRiverMetadata, observations))
	if err != nil || len(signals) != 1 {
		t.Fatalf("signals=%v error=%v", signals, err)
	}
	s := signals[0]
	if s.Value == nil || *s.Value != 6.39 || s.State != "available" || s.ApprovalStatus != "Provisional" || s.ObservationAgeSeconds == nil || *s.ObservationAgeSeconds != 3900 {
		t.Fatalf("observation lost provenance: %+v", s)
	}
	if s.Thresholds.Minor != nil || s.Thresholds.Action != nil || s.MinorStageDifference != nil || s.Relevance != "regional-context" || s.FacilityImpact != "unknown" || len(s.Gaps) < 2 {
		t.Fatalf("invented local applicability or threshold: %+v", s)
	}
	encoded, _ := json.Marshal(s)
	if !strings.Contains(string(encoded), `"minor":null`) || !strings.Contains(s.Source, "02175200") || !strings.HasSuffix(s.MetadataSource, "/srbs1") {
		t.Fatalf("missing data must serialize as null with original source: %s", encoded)
	}
}

func TestFloridaRiverComparesOnlySameResponseCompatibleUnits(t *testing.T) {
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "FL-DEMO-01"}, riverCheckTime, riverFixtureGetter(floridaRiverMetadata, ""))
	if err != nil || len(signals) != 1 {
		t.Fatalf("signals=%v error=%v", signals, err)
	}
	s := signals[0]
	if s.MinorStageDifference == nil || math.Abs(*s.MinorStageDifference-(-6.56)) > 1e-9 || s.Thresholds.Minor == nil || *s.Thresholds.Minor != 19.5 || s.ReportedCategory != "no_flooding" || s.FacilityImpact != "unknown" {
		t.Fatalf("comparison or safety boundary incorrect: %+v", s)
	}
	for _, metadata := range []string{
		strings.Replace(floridaRiverMetadata, `"primaryUnit":"ft"`, `"primaryUnit":"m"`, 1),
		strings.Replace(floridaRiverMetadata, `"stageUnits":"ft"`, `"stageUnits":"m"`, 1),
	} {
		signals, err := fetchRiverSignals(context.Background(), Facility{ID: "FL-DEMO-01"}, riverCheckTime, riverFixtureGetter(metadata, ""))
		if err != nil || signals[0].MinorStageDifference != nil {
			t.Fatalf("incompatible units were compared: %+v error=%v", signals, err)
		}
	}
}

func TestRiverMissingValuesDoNotBecomeZero(t *testing.T) {
	for _, v := range []float64{-999, -9999, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if riverNumber(&v) != nil {
			t.Errorf("invalid value %v retained", v)
		}
	}
	zero, negative := 0.0, -0.5
	if riverNumber(nil) != nil || riverNumber(&zero) == nil || *riverNumber(&zero) != 0 || riverNumber(&negative) == nil {
		t.Fatal("null must remain missing; real zero/negative stages must remain values")
	}
	metadata := strings.Replace(floridaRiverMetadata, `"primary":12.94`, `"primary":null`, 1)
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "FL-DEMO-01"}, riverCheckTime, riverFixtureGetter(metadata, ""))
	if err != nil || signals[0].State != "unknown" || signals[0].Value != nil || signals[0].MinorStageDifference != nil {
		t.Fatalf("missing observation became actionable: %+v error=%v", signals, err)
	}
}

func TestRiverDoesNotReuseOlderValueWhenLatestIsInvalid(t *testing.T) {
	older := riverUSGSFeature("USGS-02175200", "00065", "2026-10-10T22:00:00Z", `"6.39"`, "ft")
	for name, latest := range map[string]string{
		"missing value": riverUSGSFeature("USGS-02175200", "00065", "2026-10-10T23:30:00Z", `null`, "ft"),
		"sentinel":      riverUSGSFeature("USGS-02175200", "00065", "2026-10-10T23:30:00Z", `"-9999"`, "ft"),
		"future time":   riverUSGSFeature("USGS-02175200", "00065", "2026-10-11T01:30:00Z", `"6.4"`, "ft"),
	} {
		t.Run(name, func(t *testing.T) {
			observations := `{"features":[` + older + "," + latest + `]}`
			signals, err := fetchRiverSignals(context.Background(), Facility{ID: "SC-DEMO-01"}, riverCheckTime, riverFixtureGetter(barnwellRiverMetadata, observations))
			if err != nil || signals[0].Value != nil || signals[0].State != "unknown" {
				t.Fatalf("invalid latest observation reused an older value: %+v error=%v", signals, err)
			}
		})
	}
}

func TestRiverRejectsIdentityMismatchAndEmptyResults(t *testing.T) {
	for name, observations := range map[string]string{
		"empty":         `{"features":[]}`,
		"other station": `{"features":[` + riverUSGSFeature("USGS-02175500", "00065", "2026-10-10T23:30:00Z", `"2.3"`, "ft") + `]}`,
		"other datum":   `{"features":[` + riverUSGSFeature("USGS-02175200", "63160", "2026-10-10T23:30:00Z", `"15.8"`, "ft") + `]}`,
		"invalid time":  `{"features":[` + riverUSGSFeature("USGS-02175200", "00065", "not-a-time", `"6.39"`, "ft") + `]}`,
	} {
		t.Run(name, func(t *testing.T) {
			signals, err := fetchRiverSignals(context.Background(), Facility{ID: "SC-DEMO-01"}, riverCheckTime, riverFixtureGetter(barnwellRiverMetadata, observations))
			if err != nil || signals[0].Value != nil || signals[0].State != "unknown" {
				t.Fatalf("unmatched data was used: %+v error=%v", signals, err)
			}
		})
	}
	wrongMetadata := strings.Replace(floridaRiverMetadata, `"lid":"STEF1"`, `"lid":"MILS1"`, 1)
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "FL-DEMO-01"}, riverCheckTime, riverFixtureGetter(wrongMetadata, ""))
	if err == nil || signals[0].Value != nil || signals[0].Thresholds.Minor != nil {
		t.Fatalf("wrong gauge was trusted: %+v error=%v", signals, err)
	}
}

func TestRiverOldObservationReportsAgeWithoutPretendingItIsFresh(t *testing.T) {
	observations := `{"features":[` + riverUSGSFeature("USGS-02175200", "00065", "2025-12-02T23:30:00Z", `"2.42"`, "ft") + `]}`
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "SC-DEMO-01"}, riverCheckTime, riverFixtureGetter(barnwellRiverMetadata, observations))
	if err != nil || signals[0].ObservationAgeSeconds == nil || *signals[0].ObservationAgeSeconds < 86400 || signals[0].ObservedAt != "2025-12-02T23:30:00Z" || signals[0].FetchedAt != riverCheckTime.Format(time.RFC3339) {
		t.Fatalf("old observation hidden behind refresh time: %+v error=%v", signals, err)
	}
}

func TestRiverRefreshFailureAndUnconfiguredFacility(t *testing.T) {
	fail := func(context.Context, string, any) error { return context.DeadlineExceeded }
	signals, err := fetchRiverSignals(context.Background(), Facility{ID: "SC-DEMO-01"}, riverCheckTime, fail)
	if !errors.Is(err, context.DeadlineExceeded) || len(signals) != 1 || signals[0].State != "unknown" || signals[0].Value != nil || signals[0].FetchedAt != "" || signals[0].AttemptedAt == "" {
		t.Fatalf("failed refresh became evidence: %+v error=%v", signals, err)
	}
	called := false
	signals, err = fetchRiverSignals(context.Background(), Facility{ID: "unconfigured"}, riverCheckTime, func(context.Context, string, any) error { called = true; return nil })
	if err != nil || called || len(signals) != 0 {
		t.Fatalf("unconfigured facility borrowed a regional source: %+v error=%v", signals, err)
	}
}

func TestRiverHTTPRespectsCancellationAndStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Accept") != "application/json" {
			t.Error("missing public API client headers")
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	var output map[string]any
	if err := getRiverJSON(context.Background(), server.URL, &output); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("upstream error ignored: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := getRiverJSON(ctx, server.URL, &output); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}
