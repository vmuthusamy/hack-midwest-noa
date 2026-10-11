package weatherdesk

// Independent QA uses temporary persistence roots and injected NOAA evidence.
// No hosted inference, deployed API, or real recipient is contacted.
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func operationsQAApp(t *testing.T) *App {
	t.Helper()
	a, err := New(t.TempDir(), "127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.auth = qaAuthorization()
	return a
}
func operationsQARequest(t *testing.T, a *App, method, path string, body any, want int) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	p := qaPrincipal(method, path, raw)
	p.Nonce = newCaseID()
	r := qaSignedRequest(p, method, path, raw)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s got %d want %d: %s", method, path, w.Code, want, w.Body.String())
	}
	return w
}
func operationsQACase(t *testing.T, w *httptest.ResponseRecorder) *OperationsCase {
	t.Helper()
	var v struct {
		Case *OperationsCase `json:"case"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.Case == nil {
		t.Fatalf("case response %s: %v", w.Body.String(), err)
	}
	return v.Case
}
func operationsQAOpen(t *testing.T, a *App, mode string) *OperationsCase {
	return operationsQACase(t, operationsQARequest(t, a, "POST", "/api/case?facility=barnwell", map[string]string{"mode": mode}, 200))
}
func operationsQAAction(t *testing.T, a *App, c *OperationsCase, action string, extra map[string]any, want int) *httptest.ResponseRecorder {
	t.Helper()
	input := map[string]any{"action": action, "revision": c.Revision}
	for k, v := range extra {
		input[k] = v
	}
	return operationsQARequest(t, a, "POST", "/api/case/action?facility=barnwell", input, want)
}
func operationsQALive(t *testing.T, a *App) Evidence {
	t.Helper()
	replay := a.replays["barnwell"]
	now := time.Now().UTC()
	alert := replay.Alert
	alert.Issued = now.Add(-time.Minute).Format(time.RFC3339)
	alert.Effective = alert.Issued
	alert.Expires = now.Add(time.Hour).Format(time.RFC3339)
	alert.Lifecycle, alert.Geography, alert.Applicability = "active", "applicable", "applicable"
	alert.Status, alert.MessageType = "Actual", "Alert"
	return Evidence{Mode: "LIVE", Facility: replay.Facility, LastRefresh: now.Format(time.RFC3339), Source: "https://api.weather.gov/alerts/active", Alerts: []Alert{alert}, RiverSignals: []RiverSignal{}, PostureAssessment: AssessPosture("LIVE", replay.Facility, []Alert{alert}, true)}
}
func operationsQAImmutableRecord(record DecisionRecord) string {
	record.Intents = nil
	record.Rechecks = nil
	return digest(record)
}

func operationsQAManager(c *OperationsCase) Principal {
	return Principal{Sub: "google:qa123", Name: "Synthetic QA manager", Role: "manager", Facilities: []string{c.Facility.ID}}
}

func TestOperationsQAExactReviewedPackageAndRecipientRouting(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQAOpen(t, a, "REPLAY")
	if c.Mode != "REPLAY" || c.Evidence.Mode != "REPLAY" || c.EffectiveAt != c.Evidence.LastRefresh {
		t.Fatal("replay clock/mode mixed")
	}
	if fresh, resolved := caseEvidenceReady(c, time.Now().Add(24*time.Hour)); !fresh || !resolved {
		t.Fatal("replay incorrectly uses current wall clock")
	}
	operationsQAAction(t, a, c, "simulate", nil, 409)
	firstID := c.Recipients[0].ID
	operationsQAAction(t, a, c, "approve", map[string]any{"messages": map[string]string{firstID: "Unreviewed simultaneous change"}}, 409)
	effective := "2026-10-10T22:00:00Z"
	rationale := "Synthetic manager chose closure for operating review."
	c = operationsQACase(t, operationsQAAction(t, a, c, "choose", map[string]any{"operatingPosture": "closure", "effectiveAt": effective, "rationale": rationale, "messages": map[string]string{firstID: "Exact reviewed individual closure message."}}, 200))
	for _, p := range c.Recipients {
		if p.Group == "remote" && (p.Included || p.Text != "" || p.Reason == "") {
			t.Fatal("remote exclusion lost reason")
		}
		if p.Essential && (!strings.Contains(p.Text, "does not authorize travel") || strings.Contains(p.Text, "report now")) {
			t.Fatal("essential implied travel")
		}
	}
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	binding := c.Approval.Binding
	if len(c.RecordIDs) != 1 || c.Approval.Package.Parameters["rationale"] != rationale || c.Approval.Package.Parameters["effectiveAt"] != effective || c.Approval.Package.Parameters["operatingPosture"] != "closure" {
		t.Fatal("approval omitted exact decision parameters")
	}
	for _, person := range c.Recipients {
		if strings.Contains(person.Name, "Jordan") {
			email := false
			for _, channel := range person.Channels {
				if channel == "email" {
					email = true
				}
			}
			if !email {
				t.Fatal("plain-text email accessibility route missing")
			}
			var bound []CaseRecipient
			if err := json.Unmarshal([]byte(c.Approval.Package.Parameters["recipients"]), &bound); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, recipient := range bound {
				if recipient.ID == person.ID {
					found = digest(recipient.Channels) == digest(person.Channels)
				}
			}
			if !found {
				t.Fatal("approved package omitted exact planned accessibility channels")
			}
		}
	}
	if binding != digest(operationsPackage(c)) {
		t.Fatal("approval binding disagrees with reviewed package")
	}
	for _, r := range c.Recipients {
		if r.ID == firstID && r.Text != "Exact reviewed individual closure message." {
			t.Fatal("message changed at approval")
		}
	}
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	if len(c.RecordIDs) != 1 {
		t.Fatal("repeat approval duplicated record")
	}
	changes := []struct {
		name string
		edit func(*OperationsCase)
	}{
		{"rationale", func(c *OperationsCase) { c.Rationale += " Changed." }},
		{"effective time", func(c *OperationsCase) { c.EffectiveAt = "2026-10-10T23:00:00Z" }},
		{"posture", func(c *OperationsCase) { c.OperatingPosture = "delay" }},
		{"text", func(c *OperationsCase) { c.Recipients[0].Text += " Changed." }},
		{"channels", func(c *OperationsCase) { c.Recipients[0].Channels = []string{"changed-channel"} }},
		{"essential assignment", func(c *OperationsCase) { c.Recipients[0].Essential = !c.Recipients[0].Essential }},
		{"recipient selection", func(c *OperationsCase) { c.Recipients[0].Included = false }},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			changed := cloneOperations(c)
			change.edit(changed)
			if _, err := changed.queue(operationsQAManager(changed)); err == nil || len(changed.Intents) > 0 {
				t.Fatal("changed approved package queued")
			}
		})
	}
	oldRecord := a.records[c.RecordIDs[0]]
	frozen := digest(oldRecord)
	c = operationsQACase(t, operationsQAAction(t, a, c, "choose", map[string]any{"messages": map[string]string{firstID: "Changed reviewed text requires another approval."}}, 200))
	if c.Approval != nil || digest(a.records[oldRecord.ID]) != frozen {
		t.Fatal("edit retained authorization or rewrote original record")
	}
	operationsQAAction(t, a, c, "simulate", nil, 409)
}

func TestOperationsQALiveRecheckIrrelevantChangeAndDuplicateQueue(t *testing.T) {
	a := operationsQAApp(t)
	e := operationsQALive(t, a)
	calls := 0
	a.caseFetch = func(context.Context, Facility) (Evidence, error) { calls++; return e, nil }
	c := operationsQAOpen(t, a, "LIVE")
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	original := c.Approval.Binding
	outside := e.Alerts[0]
	outside.ID = "urn:qa:unrelated"
	outside.Geography = "outside"
	outside.Applicability = "not-applicable"
	outside.Event = "Red Flag Warning"
	e.Alerts = append([]Alert{outside}, e.Alerts...)
	e.LastRefresh = time.Now().Add(time.Second).UTC().Format(time.RFC3339)
	e.FeedUpdated = "irrelevant update"
	c = operationsQACase(t, operationsQAAction(t, a, c, "simulate", nil, 200))
	if calls != 2 || c.Approval.Binding != original || len(c.Intents) != 5 {
		t.Fatalf("live recheck/irrelevant behavior: calls %d intents %d", calls, len(c.Intents))
	}
	for _, i := range c.Intents {
		if i.State != "queued" || !i.Simulated || i.ApprovalBinding != original {
			t.Fatal("simulation implied delivery or lost authorization")
		}
	}
	c = operationsQACase(t, operationsQAAction(t, a, c, "simulate", nil, 200))
	if calls != 3 || len(c.Intents) != 5 {
		t.Fatal("repeat simulation did not refetch or duplicated intents")
	}
}

func TestOperationsQAMaterialEvidenceAndFetchFailureBlockQueue(t *testing.T) {
	for _, kind := range []string{"warning update", "expired", "unknown applicability", "failed fetch"} {
		t.Run(kind, func(t *testing.T) {
			a := operationsQAApp(t)
			e := operationsQALive(t, a)
			fail := false
			a.caseFetch = func(context.Context, Facility) (Evidence, error) {
				if fail {
					return Evidence{}, errors.New("synthetic failure")
				}
				return e, nil
			}
			c := operationsQAOpen(t, a, "LIVE")
			c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
			old := operationsQAImmutableRecord(a.records[c.RecordIDs[0]])
			switch kind {
			case "warning update":
				e.Alerts[0].MessageType = "Update"
				e.Alerts[0].Description += " Material change."
			case "expired":
				e.Alerts[0].Expires = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
				e.Alerts[0].Lifecycle = "expired"
				e.Alerts[0].Applicability = "not-applicable"
			case "unknown applicability":
				e.Alerts[0].Geography = "unknown"
				e.Alerts[0].Applicability = "unknown"
			case "failed fetch":
				fail = true
			}
			status := 409
			if fail {
				status = 502
			}
			operationsQAAction(t, a, c, "simulate", nil, status)
			current := a.cases["barnwell"]
			if len(current.Intents) != 0 || operationsQAImmutableRecord(a.records[c.RecordIDs[0]]) != old {
				t.Fatal("blocked recheck queued or rewrote approved record")
			}
			if !fail && current.Approval != nil {
				t.Fatal("material evidence did not invalidate approval")
			}
		})
	}
}

func TestOperationsQAMissingStaleAndUnknownEvidence(t *testing.T) {
	for _, kind := range []string{"missing refresh", "stale refresh", "future refresh", "unknown candidate", "wrong mode", "wrong facility", "expiry elapsed"} {
		t.Run(kind, func(t *testing.T) {
			a := operationsQAApp(t)
			e := operationsQALive(t, a)
			switch kind {
			case "missing refresh":
				e.LastRefresh = ""
			case "stale refresh":
				e.LastRefresh = time.Now().Add(-6 * time.Minute).UTC().Format(time.RFC3339)
			case "future refresh":
				e.LastRefresh = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
			case "unknown candidate":
				e.Alerts[0].Geography = "unknown"
				e.Alerts[0].Applicability = "unknown"
			case "wrong mode":
				e.Mode = "REPLAY"
			case "wrong facility":
				e.Facility.ID = "OTHER"
			case "expiry elapsed":
				e.Alerts[0].Expires = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			}
			// Build a consistent case identity, then inject malformed evidence at readiness boundary.
			c := &OperationsCase{Mode: "LIVE", Facility: a.replays["barnwell"].Facility, Evidence: e, OperatingPosture: "delay", PolicyVersion: operationsPolicyVersion}
			if fresh, resolved := caseEvidenceReady(c, time.Now()); fresh && resolved {
				t.Fatal("missing/stale evidence accepted")
			}
			if _, err := c.approve(operationsQAManager(c)); err == nil {
				t.Fatal("incomplete evidence approved")
			}
		})
	}
}

func TestOperationsQAScopedManagerRequiredAtHTTPAndPolicy(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQAOpen(t, a, "REPLAY")
	a.auth = qaAuthorization()
	for _, kind := range []string{"viewer", "other facility"} {
		t.Run(kind, func(t *testing.T) {
			path := "/api/case/action?facility=barnwell"
			body, _ := json.Marshal(map[string]string{"action": "approve", "revision": c.Revision})
			p := qaPrincipal("POST", path, body)
			p.Nonce = newCaseID()
			if kind == "viewer" {
				p.Role = "viewer"
			} else {
				p.Facilities = []string{"FL-DEMO-01"}
			}
			before := digest(a.cases["barnwell"])
			w := httptest.NewRecorder()
			a.ServeHTTP(w, qaSignedRequest(p, "POST", path, body))
			if w.Code != 403 || digest(a.cases["barnwell"]) != before {
				t.Fatal("unauthorized manager/facility changed case", w.Code)
			}
			if _, err := cloneOperations(c).approve(p); err == nil {
				t.Fatal("policy widened principal authority")
			}
		})
	}
	path := "/api/case?facility=barnwell"
	p := qaPrincipal("GET", path, nil)
	p.Role = "viewer"
	w := httptest.NewRecorder()
	a.ServeHTTP(w, qaSignedRequest(p, "GET", path, nil))
	if w.Code != 200 {
		t.Fatal("scoped viewer could not inspect case")
	}
}

func TestOperationsQARecordArchivesSimulationAndSurvivesRestart(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQAOpen(t, a, "REPLAY")
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	recordID := c.RecordIDs[0]
	approved := a.records[recordID].Approval
	c = operationsQACase(t, operationsQAAction(t, a, c, "simulate", nil, 200))
	operationsQAOpen(t, a, "REPLAY")
	record, ok := a.recordFor(recordID, c.Facility.ID)
	if !ok || len(record.Intents) != 5 || len(record.Rechecks) == 0 {
		t.Fatalf("opening new case lost prior simulated notice record: intents %d rechecks %d", len(record.Intents), len(record.Rechecks))
	}
	restarted, err := New(a.root, "127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	restored, ok := restarted.recordFor(recordID, c.Facility.ID)
	if !ok || len(restored.Intents) != 5 || digest(restored.Approval) != digest(approved) {
		t.Fatal("restart lost immutable approval/simulation record")
	}
	if _, ok := restarted.recordFor(recordID, "FL-DEMO-01"); ok {
		t.Fatal("record exposed across facilities")
	}
}

func TestOperationsQAFailedPersistenceRollsBackCaseAndRecord(t *testing.T) {
	for _, action := range []string{"approve", "simulate"} {
		t.Run(action, func(t *testing.T) {
			a := operationsQAApp(t)
			c := operationsQAOpen(t, a, "REPLAY")
			if action == "simulate" {
				c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
			}
			beforeCase, beforeRecords := digest(a.cases), digest(a.records)
			if err := os.RemoveAll(filepath.Join(a.root, ".cache")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(a.root, ".cache"), []byte("blocked persistence"), 0600); err != nil {
				t.Fatal(err)
			}
			operationsQAAction(t, a, c, action, nil, 500)
			if digest(a.cases) != beforeCase || digest(a.records) != beforeRecords {
				t.Fatal("failed persistence published approval, queue, or record changes")
			}
		})
	}
}

func TestOperationsQAConcurrentDraftChangeDuringLiveRecheck(t *testing.T) {
	a := operationsQAApp(t)
	e := operationsQALive(t, a)
	a.caseFetch = func(context.Context, Facility) (Evidence, error) { return e, nil }
	c := operationsQAOpen(t, a, "LIVE")
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	a.caseFetch = func(context.Context, Facility) (Evidence, error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		edited := cloneOperations(a.cases["barnwell"])
		if err := edited.choose("", "New concurrent manager rationale.", "", nil, operationsQAManager(edited)); err != nil {
			t.Fatal(err)
		}
		a.cases["barnwell"] = edited
		return e, nil
	}
	operationsQAAction(t, a, c, "simulate", nil, 409)
	if len(a.cases["barnwell"].Intents) > 0 {
		t.Fatal("racing stale approval queued")
	}
}

func TestOperationsQARechecksBelongToExactApprovalRecord(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQAOpen(t, a, "REPLAY")
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	first := c.RecordIDs[0]
	c = operationsQACase(t, operationsQAAction(t, a, c, "simulate", nil, 200))
	c = operationsQACase(t, operationsQAAction(t, a, c, "choose", map[string]any{"rationale": "Second independently reviewed operating rationale."}, 200))
	c = operationsQACase(t, operationsQAAction(t, a, c, "approve", nil, 200))
	second := c.RecordIDs[1]
	c = operationsQACase(t, operationsQAAction(t, a, c, "simulate", nil, 200))
	operationsQAAction(t, a, c, "simulate", nil, 200)
	firstRecord, _ := a.recordFor(first, c.Facility.ID)
	secondRecord, _ := a.recordFor(second, c.Facility.ID)
	if len(firstRecord.Intents) != 5 || len(secondRecord.Intents) != 5 {
		t.Fatal("notices mixed between approvals")
	}
	if len(firstRecord.Rechecks) != 1 || len(secondRecord.Rechecks) != 2 {
		t.Fatalf("rechecks attributed by case/time rather than approval: first %d second %d", len(firstRecord.Rechecks), len(secondRecord.Rechecks))
	}
}

// Opt-in browser fixture server: never started by ordinary unit/race runs.
func TestBrowserCaseQAFixtureServer(t *testing.T) {
	if os.Getenv("WEATHER_DESK_BROWSER_QA_SERVER") != "true" {
		t.Skip("isolated browser fixture server opt-in only")
	}
	address := os.Getenv("WEATHER_DESK_HTTP_ADDR")
	if address != "127.0.0.1:8093" {
		t.Fatal("browser fixture server requires isolated8093")
	}
	if os.Getenv("AUTH_REQUIRED") != "true" {
		t.Fatal("browser QA requires actual signed-principal middleware")
	}
	a, err := New(t.TempDir(), "127.0.0.1:9999")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.caseFetch = func(_ context.Context, f Facility) (Evidence, error) {
		scenario := ""
		for key, r := range a.replays {
			if r.Facility.ID == f.ID {
				scenario = key
			}
		}
		if scenario == "" {
			return Evidence{}, errors.New("fixture facility missing")
		}
		e, err := recordedCaseEvidence(scenario)
		if err != nil {
			return e, err
		}
		now := time.Now().UTC()
		e.Mode = "LIVE"
		e.LastRefresh = now.Format(time.RFC3339)
		e.Gaps = append(e.Gaps, "QA FIXTURE: recorded NOAA input retimed only for local UI tests; not current conditions.")
		for i := range e.Alerts {
			e.Alerts[i].Issued = now.Add(-time.Minute).Format(time.RFC3339)
			e.Alerts[i].Effective = e.Alerts[i].Issued
			e.Alerts[i].Expires = now.Add(30 * time.Minute).Format(time.RFC3339)
			e.Alerts[i].Lifecycle = "active"
			if e.Alerts[i].Geography == "applicable" {
				e.Alerts[i].Applicability = "applicable"
			}
		}
		e.PostureAssessment = AssessPosture("LIVE", f, e.Alerts, true)
		return e, nil
	}
	server := &http.Server{Addr: address, Handler: a, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	t.Log("Isolated signed Go fixture server ready on8093; all weather/delivery inputs are fixtures.")
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatal(err)
		}
	case <-ctx.Done():
	case <-time.After(15 * time.Minute):
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server.Shutdown(shutdown)
}

func TestOperationsQARecordedBatesburgAndBarnwellGeographicSeparation(t *testing.T) {
	const warningID = "urn:oid:2.49.0.1.840.0.be5613bdb2f88e91a29b7e41456b681f8ab18db1.001.1"
	batesburg, err := recordedCaseEvidence("batesburg")
	if err != nil {
		t.Fatal(err)
	}
	if batesburg.Mode != "REPLAY" || batesburg.LastRefresh != "2026-10-11T02:20:00Z" || batesburg.Facility.ID != "SC-DEMO-02" || batesburg.Facility.Latitude != 33.91 || batesburg.Facility.Longitude != -81.54 || !batesburg.Facility.Fictional {
		t.Fatal("recorded facility or capture metadata changed")
	}
	if len(batesburg.Alerts) != 1 {
		t.Fatal("Batesburg warning absent or duplicated")
	}
	warning := batesburg.Alerts[0]
	if warning.ID != warningID || warning.Event != "Flash Flood Warning" || warning.Geography != "applicable" || warning.Applicability != "applicable" || warning.Lifecycle != "active" {
		t.Fatalf("captured point-in-polygon applicability wrong: %+v", warning)
	}
	assessment := batesburg.PostureAssessment
	if assessment.RecommendedPosture == nil || *assessment.RecommendedPosture != "telework" || !assessment.Illustrative || assessment.Mode != "REPLAY" {
		t.Fatal("captured local warning did not produce illustrative telework review")
	}
	found := false
	for _, trigger := range assessment.Triggers {
		if trigger.AlertID == warningID && trigger.Matched {
			found = true
		}
	}
	if !found {
		t.Fatal("recommendation missing exact recorded warning provenance")
	}
	barnwell, err := recordedCaseEvidence("barnwell")
	if err != nil {
		t.Fatal(err)
	}
	if barnwell.LastRefresh != batesburg.LastRefresh || barnwell.PostureAssessment.Status != "planning-review" || barnwell.PostureAssessment.RecommendedPosture != nil {
		t.Fatal("outside Barnwell acquired warning posture rather than watch planning review")
	}
	for _, alert := range barnwell.Alerts {
		if alert.ID == warningID {
			t.Fatal("outside warning entered local policy evidence")
		}
	}
	if len(barnwell.NearbyAlerts) != 1 || barnwell.NearbyAlerts[0].ID != warningID || barnwell.NearbyAlerts[0].Geography != "outside" || barnwell.NearbyAlerts[0].Applicability != "not-applicable" {
		t.Fatal("map-only warning did not preserve outside classification")
	}
	if len(barnwell.RiverSignals) != 1 || barnwell.RiverSignals[0].Thresholds.Minor != nil || barnwell.RiverSignals[0].FacilityImpact != "unknown" {
		t.Fatal("regional gauge invented flood threshold/facility impact")
	}
}

func TestOperationsQARecordedBatesburgFirstCaseHasNoPriorApproval(t *testing.T) {
	a := operationsQAApp(t)
	const path = "/api/case?facility=batesburg"
	raw := []byte(`{"mode":"REPLAY"}`)
	principal := qaPrincipal("POST", path, raw)
	principal.Facilities = []string{"SC-DEMO-02"}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, qaSignedRequest(principal, "POST", path, raw))
	if w.Code != 200 {
		t.Fatalf("recorded case: %d %s", w.Code, w.Body.String())
	}
	c := operationsQACase(t, w)
	if c.Approval != nil || len(c.RecordIDs) != 0 || len(c.Intents) != 0 || c.OperatingPosture != "telework" || c.EffectiveAt != "2026-10-11T02:20:00Z" {
		t.Fatal("first recorded case invented prior approval/intents or used live clock")
	}
	if len(c.History) != 1 || c.History[0].Label != "Decision case opened" {
		t.Fatalf("invented prior notice history: %+v", c.History)
	}
	for _, person := range c.Recipients {
		if strings.Contains(person.Text, "Report at 10:00") || (person.Essential && strings.Contains(strings.ToLower(person.Text), "report now")) {
			t.Fatal("new case invented prior instruction or essential travel directive")
		}
	}
	if fresh, resolved := caseEvidenceReady(c, time.Now().Add(365*24*time.Hour)); !fresh || !resolved {
		t.Fatal("recorded capture is wrongly projected against future wall clock")
	}
	if path := os.Getenv("WEATHER_DESK_BATESBURG_QA_EXPORT"); path != "" {
		raw, err := json.MarshalIndent(map[string]any{"mode": c.Mode, "facility": c.Facility, "evidence": c.Evidence, "roster": c.Roster, "recipients": c.Recipients, "operatingPosture": c.OperatingPosture, "rationale": c.Rationale, "effectiveAt": c.EffectiveAt, "nextUpdateAt": c.NextUpdateAt, "revision": c.Revision, "conditionsDigest": caseConditions(c.Evidence), "approval": c.Approval}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func operationsQABatesburgRequest(t *testing.T, a *App, endpoint string, input any, want int) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	path := endpoint + "?facility=batesburg"
	principal := qaPrincipal("POST", path, raw)
	principal.Facilities = []string{"SC-DEMO-02"}
	principal.Nonce = newCaseID()
	w := httptest.NewRecorder()
	a.ServeHTTP(w, qaSignedRequest(principal, "POST", path, raw))
	if w.Code != want {
		t.Fatalf("%s %d want%d: %s", endpoint, w.Code, want, w.Body.String())
	}
	return w
}
func TestOperationsQAPowerChangesRequireExactReviewAndPreserveApprovalRecord(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case", map[string]string{"mode": "REPLAY"}, 200))
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 200))
	recordID := c.RecordIDs[0]
	original := digest(a.records[recordID])
	binding := c.Approval.Binding
	revision := c.Revision
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]any{"action": "power", "revision": revision, "powerSignal": PowerSignal{Status: "yes", Source: "Synthetic onsite manager report", ObservedAt: "2026-10-11T02:19:00Z"}}, 200))
	if c.Approval != nil || c.Revision == revision || digest(a.records[recordID]) != original || c.Evidence.PowerSignal.Mode != "EXAMPLE" || c.Evidence.PostureAssessment.RecommendedPosture == nil || *c.Evidence.PostureAssessment.RecommendedPosture != "closure" {
		t.Fatal("power change failed to invalidate exact approval or rewrote original record")
	}
	if c.OperatingPosture != "telework" {
		t.Fatal("suggestion silently chose manager posture")
	}
	operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "simulate", "revision": c.Revision}, 409)
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "choose", "revision": c.Revision, "operatingPosture": "closure", "rationale": "Manager reviewed recorded warning and example power outage."}, 200))
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 200))
	if c.Approval.Binding == binding || !strings.Contains(c.Approval.Package.Parameters["powerSignal"], "EXAMPLE") || c.Approval.Package.Parameters["nextUpdateAt"] != "2026-10-11T06:00:00-04:00" || len(c.RecordIDs) != 2 {
		t.Fatal("power/next-update omitted from changed approval binding")
	}
	for _, person := range c.Recipients {
		if person.Included && !strings.Contains(person.Text, "Next update Oct 11, 6:00 AM facility time") {
			t.Fatal("bound notification omitted exact next-update time")
		}
	}
	// Outside warning plus a local power report cannot acquire a warning-based closure suggestion.
	watch, err := recordedCaseEvidence("barnwell")
	if err != nil {
		t.Fatal(err)
	}
	watch.PowerSignal = &PowerSignal{Status: "yes", Source: "Synthetic QA report", ObservedAt: watch.LastRefresh, Mode: "EXAMPLE"}
	applyPowerAssessment(&watch)
	if watch.PostureAssessment.RecommendedPosture != nil {
		t.Fatal("outside map-only warning promoted to power closure")
	}
}
func TestOperationsQAInvalidPowerReportsDoNotMutateCase(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case", map[string]string{"mode": "REPLAY"}, 200))
	original := digest(a.cases["batesburg"])
	for _, report := range []*PowerSignal{nil, {Status: "maybe", Source: "Synthetic QA", ObservedAt: c.Evidence.LastRefresh}, {Status: "yes", Source: "", ObservedAt: c.Evidence.LastRefresh}, {Status: "yes", Source: "Synthetic QA", ObservedAt: "unknown"}, {Status: "yes", Source: "Synthetic QA", ObservedAt: "2026-10-11T02:22:00Z"}} {
		operationsQABatesburgRequest(t, a, "/api/case/action", map[string]any{"action": "power", "revision": c.Revision, "powerSignal": report}, 409)
		if digest(a.cases["batesburg"]) != original {
			t.Fatal("invalid power report published state")
		}
	}
}
func TestOperationsQADeliveryExamplesSeparateFromQueuedIntentsAndSurviveRestart(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case", map[string]string{"mode": "REPLAY"}, 200))
	operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "delivery-fixture", "revision": c.Revision}, 409)
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 200))
	id := c.RecordIDs[0]
	approval := digest(c.Approval)
	operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "delivery-fixture", "revision": c.Revision}, 409)
	c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "simulate", "revision": c.Revision}, 200))
	queued := digest(c.Intents)
	w := operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "delivery-fixture", "revision": c.Revision}, 200)
	c = operationsQACase(t, w)
	var response struct {
		Added int `json:"added"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Added != 17 || len(c.DeliveryEvents) != 17 || digest(c.Intents) != queued || digest(c.Approval) != approval {
		t.Fatal("fixture changed original queued intents/approval or wrong event count")
	}
	unreachable, nextChannel, acknowledged := 0, 0, 0
	for _, event := range c.DeliveryEvents {
		if !event.Simulated || event.ApprovalBinding != c.Approval.Binding || !strings.Contains(event.Detail, "no network delivery") {
			t.Fatal("delivery fixture lost simulation provenance")
		}
		switch event.State {
		case "unreachable":
			unreachable++
		case "next-channel":
			nextChannel++
		case "acknowledged":
			acknowledged++
		}
		if event.Recipient == "S02" && event.State == "next-channel" && event.Channel == c.Intents[1].Channels[0] {
			t.Fatal("fallback reused first channel")
		}
	}
	if unreachable != 1 || nextChannel != 1 || acknowledged != 4 {
		t.Fatal("fixture unreachable/fallback/acknowledgment outcomes incorrect")
	}
	w = operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "delivery-fixture", "revision": c.Revision}, 200)
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Added != 0 || len(operationsQACase(t, w).DeliveryEvents) != 17 {
		t.Fatal("repeated fixture duplicated states")
	}
	restarted, err := New(a.root, "127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	record, ok := restarted.recordFor(id, c.Facility.ID)
	if !ok || len(record.DeliveryEvents) != 17 || len(record.Intents) != 5 || digest(record.Approval) != approval {
		t.Fatal("restart lost exact approval and separate queued/delivery record")
	}
}

func TestOperationsQAPowerRecoveryCannotBypassOverrideApprovalGate(t *testing.T) {
	for _, status := range []string{"no", "unknown"} {
		t.Run(status, func(t *testing.T) {
			a := operationsQAApp(t)
			c := operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case", map[string]string{"mode": "REPLAY"}, 200))
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]any{"action": "power", "revision": c.Revision, "powerSignal": PowerSignal{Status: "yes", Source: "Synthetic manager", ObservedAt: c.Evidence.LastRefresh}}, 200))
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "choose", "revision": c.Revision, "operatingPosture": "closure"}, 200))
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 200))
			recordID := c.RecordIDs[0]
			original := digest(a.records[recordID])
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]any{"action": "power", "revision": c.Revision, "powerSignal": PowerSignal{Status: status, Source: "Synthetic manager recheck", ObservedAt: c.Evidence.LastRefresh}}, 200))
			if c.Approval != nil || c.OperatingPosture != "closure" || c.Evidence.PostureAssessment.RecommendedPosture == nil || *c.Evidence.PostureAssessment.RecommendedPosture != "telework" {
				t.Fatal("power recovery failed to restore warning-only suggestion and invalidate approval")
			}
			before := digest(a.cases["batesburg"])
			operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 409)
			if digest(a.cases["batesburg"]) != before || digest(a.records[recordID]) != original {
				t.Fatal("blocked direct approval mutated prior record")
			}
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "choose", "revision": c.Revision, "operatingPosture": "closure", "overrideReason": "Manager reviewed local conditions and retains closure."}, 200))
			c = operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case/action", map[string]string{"action": "approve", "revision": c.Revision}, 200))
			if c.Approval.Package.Parameters["overrideReason"] == "" {
				t.Fatal("reviewed override not bound")
			}
		})
	}
}
func TestOperationsQAActualRecordedBriefingCacheExactInputBinding(t *testing.T) {
	a := operationsQAApp(t)
	c := operationsQACase(t, operationsQABatesburgRequest(t, a, "/api/case", map[string]string{"mode": "REPLAY"}, 200))
	cache := c.CachedBriefing
	if cache == nil || cache.GeneratedAt == "" || cache.ModelID == "" || cache.EvidenceConditions != caseConditions(c.Evidence) || cache.EvidenceTimestamp != c.Evidence.LastRefresh || cache.Briefing["explanation"] == nil {
		t.Fatal("accepted real recorded cache absent or not exact input bound")
	}
	for _, variant := range []struct {
		name string
		edit func(*OperationsCase)
	}{
		{"mode", func(c *OperationsCase) { c.Mode = "LIVE" }}, {"facility", func(c *OperationsCase) { c.Facility.ID = "OTHER" }}, {"capture", func(c *OperationsCase) { c.Evidence.LastRefresh = "2026-10-11T02:21:00Z" }}, {"material evidence", func(c *OperationsCase) { c.Evidence.Alerts[0].Description += " Changed source evidence." }}, {"posture", func(c *OperationsCase) { c.OperatingPosture = "closure" }}, {"effective time", func(c *OperationsCase) { c.EffectiveAt = "2026-10-11T02:21:00Z" }}, {"power", func(c *OperationsCase) { c.Evidence.PowerSignal = &PowerSignal{Status: "unknown", Mode: "EXAMPLE"} }},
	} {
		t.Run(variant.name, func(t *testing.T) {
			changed := cloneOperations(c)
			changed.CachedBriefing = nil
			variant.edit(changed)
			attachCachedBriefing(changed, "batesburg")
			if changed.CachedBriefing != nil {
				t.Fatal("cached AI reused after inputs changed")
			}
		})
	}
	changed := cloneOperations(c)
	if err := changed.choose("telework", changed.Rationale, "2026-10-11T02:21:00Z", nil, operationsQAManager(changed)); err != nil {
		t.Fatal(err)
	}
	if changed.CachedBriefing != nil {
		t.Fatal("changing effective time retained cached AI rationale")
	}
	if err := c.setPower(&PowerSignal{Status: "unknown", Source: "Synthetic manager", ObservedAt: c.Evidence.LastRefresh}, operationsQAManager(c)); err != nil {
		t.Fatal(err)
	}
	if c.CachedBriefing != nil {
		t.Fatal("power evidence retained cached AI rationale")
	}
}

func TestOperationsQAStaleReviewReturnsCurrentCaseWithoutApplyingDraft(t *testing.T) {
	a := operationsQAApp(t)
	e, err := recordedCaseEvidence("barnwell")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	e.Mode = "LIVE"
	e.LastRefresh = now.Format(time.RFC3339)
	for i := range e.Alerts {
		e.Alerts[i].Issued = now.Add(-time.Minute).Format(time.RFC3339)
		e.Alerts[i].Effective = e.Alerts[i].Issued
		e.Alerts[i].Expires = now.Add(time.Hour).Format(time.RFC3339)
		e.Alerts[i].Lifecycle = "active"
	}
	e.PostureAssessment = AssessPosture("LIVE", e.Facility, e.Alerts, true)
	a.caseFetch = func(context.Context, Facility) (Evidence, error) { return e, nil }
	loaded := operationsQAOpen(t, a, "LIVE")
	newer := operationsQACase(t, operationsQAAction(t, a, loaded, "choose", map[string]any{"rationale": "Another manager refreshed the operating rationale."}, 200))
	before := digest(a.cases["barnwell"])
	for _, action := range []string{"choose", "approve", "simulate"} {
		t.Run(action, func(t *testing.T) {
			w := operationsQAAction(t, a, loaded, action, map[string]any{"operatingPosture": "closure", "rationale": "Unsaved manager closure proposal must remain only a draft."}, 409)
			current := operationsQACase(t, w)
			if current.Revision != newer.Revision || current.Facility.ID != loaded.Facility.ID || current.Mode != "LIVE" || digest(a.cases["barnwell"]) != before || current.Approval != nil || len(current.Intents) > 0 {
				t.Fatal("stale conflict applied draft, widened authority or omitted current recovery case")
			}
		})
	}
}
