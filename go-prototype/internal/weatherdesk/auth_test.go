package weatherdesk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var qaAuthSecret = []byte(strings.Repeat("q", 32))

func qaAuthorization() *authorization {
	return &authorization{required: true, secret: qaAuthSecret, nonces: map[string]int64{}}
}
func qaPrincipal(method, path string, body []byte) Principal {
	h := sha256.Sum256(body)
	return Principal{Sub: "github:123", Name: "QA manager", Login: "manager", Role: "manager", Facilities: []string{"SC-DEMO-01"}, Exp: time.Now().Unix() + 60, Method: method, Path: path, BodySHA256: hex.EncodeToString(h[:]), Nonce: "qa-independent-nonce-123456"}
}
func qaSignedRequest(p Principal, method, path string, body []byte) *http.Request {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	raw, _ := json.Marshal(p)
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, qaAuthSecret)
	mac.Write([]byte(encoded))
	r.Header.Set("X-Weather-Principal", encoded)
	r.Header.Set("X-Weather-Signature", hex.EncodeToString(mac.Sum(nil)))
	return r
}
func TestPrincipalBindsExactReviewedRequestAndRestoresBody(t *testing.T) {
	body := []byte(`{"action":"approve","messages":{"onsite":"Reviewed exact text"}}`)
	path := "/api/replay/action?facility=barnwell"
	p, err := qaAuthorization().verify(qaSignedRequest(qaPrincipal("POST", path, body), "POST", path, body), "SC-DEMO-01")
	if err != nil || p.Sub != "github:123" || p.Role != "manager" {
		t.Fatal(p, err)
	}
	r := qaSignedRequest(qaPrincipal("POST", path, body), "POST", path, body)
	_, err = qaAuthorization().verify(r, "SC-DEMO-01")
	restored, _ := io.ReadAll(r.Body)
	if err != nil || !bytes.Equal(restored, body) {
		t.Fatal("authorization changed approved bytes", err)
	}
}
func TestPrincipalRejectsChangedRequestAuthorityAndFacility(t *testing.T) {
	body := []byte(`{"action":"approve"}`)
	path := "/api/replay/action?facility=barnwell"
	cases := []struct {
		name                         string
		mutate                       func(*Principal)
		method, path, body, facility string
	}{
		{name: "expired", mutate: func(p *Principal) { p.Exp = time.Now().Unix() - 1 }},
		{name: "far future", mutate: func(p *Principal) { p.Exp = time.Now().Unix() + 600 }},
		{name: "no subject", mutate: func(p *Principal) { p.Sub = "" }},
		{name: "agent role", mutate: func(p *Principal) { p.Role = "agent" }},
		{name: "fixture cannot be OAuth", mutate: func(p *Principal) { p.Role = "fixture-manager" }},
		{name: "viewer cannot write", mutate: func(p *Principal) { p.Role = "viewer" }},
		{name: "no nonce", mutate: func(p *Principal) { p.Nonce = "" }},
		{name: "other facility scope", mutate: func(p *Principal) { p.Facilities = []string{"FL-DEMO-01"} }},
		{name: "method changed", method: "GET"}, {name: "facility query changed", path: "/api/replay/action?facility=florida"}, {name: "message changed", body: `{"action":"approve","messages":"edited"}`}, {name: "selected facility changed", facility: "FL-DEMO-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := qaPrincipal("POST", path, body)
			if c.mutate != nil {
				c.mutate(&p)
			}
			method, requestPath, requestBody, facility := "POST", path, body, "SC-DEMO-01"
			if c.method != "" {
				method = c.method
			}
			if c.path != "" {
				requestPath = c.path
			}
			if c.body != "" {
				requestBody = []byte(c.body)
			}
			if c.facility != "" {
				facility = c.facility
			}
			if _, err := qaAuthorization().verify(qaSignedRequest(p, method, requestPath, requestBody), facility); err == nil {
				t.Fatal("changed authority or package accepted")
			}
		})
	}
}
func TestPrincipalRejectsForgedSignatureAndOversizedBody(t *testing.T) {
	path := "/api/replay/action?facility=barnwell"
	body := []byte("{}")
	for _, signature := range []string{"", "not-hex", strings.Repeat("a", 64)} {
		r := qaSignedRequest(qaPrincipal("POST", path, body), "POST", path, body)
		r.Header.Set("X-Weather-Signature", signature)
		if _, err := qaAuthorization().verify(r, "SC-DEMO-01"); err == nil {
			t.Fatal("forged signature accepted")
		}
	}
	large := []byte(strings.Repeat("x", 32769))
	if _, err := qaAuthorization().verify(qaSignedRequest(qaPrincipal("POST", path, large), "POST", path, large), "SC-DEMO-01"); err == nil {
		t.Fatal("oversized request accepted")
	}
}
func TestPrincipalNoncePreventsRepeatedWriteButViewerCanRead(t *testing.T) {
	a := qaAuthorization()
	path := "/api/replay/action?facility=barnwell"
	body := []byte("{}")
	p := qaPrincipal("POST", path, body)
	if _, err := a.verify(qaSignedRequest(p, "POST", path, body), "SC-DEMO-01"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.verify(qaSignedRequest(p, "POST", path, body), "SC-DEMO-01"); err == nil {
		t.Fatal("same write replayed")
	}
	readPath := "/api/replay?facility=barnwell"
	viewer := qaPrincipal("GET", readPath, nil)
	viewer.Role = "viewer"
	for i := 0; i < 2; i++ {
		if _, err := a.verify(qaSignedRequest(viewer, "GET", readPath, nil), "SC-DEMO-01"); err != nil {
			t.Fatal("viewer read denied", err)
		}
	}
}
func TestPrincipalPolicyUsesAuthenticatedRoleAndScope(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	p := Principal{Sub: "github:123", Role: "manager", Facilities: []string{"SC-DEMO-01"}}
	input := policyInputFor(r, "approve", p)
	if input["actor_role"] != "manager" || input["actor_facility"] != "SC-DEMO-01" {
		t.Fatal("policy ignored principal", input)
	}
	p.Role = "viewer"
	p.Facilities = []string{"FL-DEMO-01"}
	input = policyInputFor(r, "approve", p)
	if input["actor_role"] != "viewer" || input["actor_facility"] != "" {
		t.Fatal("policy widened scope", input)
	}
}
func TestRequiredAuthCannotStartWithoutSecret(t *testing.T) {
	t.Setenv("AUTH_REQUIRED", "true")
	for _, secret := range []string{"", strings.Repeat("x", 31)} {
		t.Setenv("WEATHER_DESK_GATEWAY_SECRET", secret)
		if _, err := newAuthorization(); err == nil {
			t.Fatal("auth requires adequate secret")
		}
	}
	t.Setenv("WEATHER_DESK_GATEWAY_SECRET", strings.Repeat("x", 32))
	if _, err := newAuthorization(); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredHTTPAuthCoversKnownAndUnknownFacility(t *testing.T) {
	app := qaBriefingApp(t, nil)
	app.auth = qaAuthorization()
	for _, path := range []string{"/api/replay?facility=barnwell", "/api/status?facility=barnwell", "/api/status?facility=unknown"} {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code == http.StatusOK {
			t.Fatalf("Unauthenticated metadata or evidence returned for %s", path)
		}
	}
	read := "/api/replay?facility=barnwell"
	p := qaPrincipal("GET", read, nil)
	p.Role = "viewer"
	w := httptest.NewRecorder()
	app.ServeHTTP(w, qaSignedRequest(p, "GET", read, nil))
	if w.Code != http.StatusOK {
		t.Fatal("Viewer cannot read scoped evidence", w.Code, w.Body.String())
	}
	path := "/api/replay/action?facility=barnwell"
	body, _ := json.Marshal(map[string]string{"action": "approve", "revision": digest(packageFor(app.replays["barnwell"]))})
	p = qaPrincipal("POST", path, body)
	p.Role = "viewer"
	before := digest(app.replays["barnwell"])
	w = httptest.NewRecorder()
	app.ServeHTTP(w, qaSignedRequest(p, "POST", path, body))
	if w.Code != http.StatusForbidden || digest(app.replays["barnwell"]) != before {
		t.Fatal("Viewer mutated saved approval", w.Code)
	}
}
func TestAuthenticatedManagerApprovalRecordsActorAndViewerCannotApprove(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	manager := Principal{Sub: "github:123", Name: "QA verified manager", Role: "manager", Facilities: []string{r.Facility.ID}}
	if err := r.approveAs(manager); err != nil {
		t.Fatal(err)
	}
	if r.Approval.ActorID != manager.Sub || r.Approval.ActorName != manager.Name || r.Approval.ApprovedAt == "" {
		t.Fatal("Approval lost authenticated actor", r.Approval)
	}
	if _, err := time.Parse(time.RFC3339, r.Approval.ApprovedAt); err != nil {
		t.Fatal("Missing approval time", err)
	}
	viewer := manager
	viewer.Role = "viewer"
	before := digest(r)
	if err := r.approveAs(viewer); err == nil || digest(r) != before {
		t.Fatal("Viewer changed approval")
	}
	if _, err := r.simulateAs(viewer); err == nil || len(r.Intents) != 0 {
		t.Fatal("Viewer simulated operational sending")
	}
	manager.Facilities = []string{"FL-DEMO-01"}
	if err := r.approveAs(manager); err == nil {
		t.Fatal("Other-facility manager approved")
	}
}

func TestPublicBindingCannotSilentlyUseDemoAuthority(t *testing.T) {
	t.Setenv("WEATHER_DESK_HTTP_ADDR", "0.0.0.0:8087")
	for _, value := range []string{"", "false", "TRUE", "typo"} {
		t.Setenv("AUTH_REQUIRED", value)
		if _, err := newAuthorization(); err == nil {
			t.Fatal("Public binding accepted implicit demo authority", value)
		}
	}
	t.Setenv("AUTH_REQUIRED", "true")
	t.Setenv("WEATHER_DESK_GATEWAY_SECRET", strings.Repeat("x", 32))
	if _, err := newAuthorization(); err != nil {
		t.Fatal("Explicit signed public backend rejected", err)
	}
}
