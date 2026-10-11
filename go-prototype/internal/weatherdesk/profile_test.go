package weatherdesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecipientProfileChangesRequireNewExactApproval(t *testing.T) {
	changes := []struct {
		name   string
		mutate func(*Replay)
	}{{"essential assignment", func(r *Replay) { r.Roster[0].Essential = !r.Roster[0].Essential }}, {"delivery channel", func(r *Replay) { r.Roster[0].Channels = []string{"sms"} }}, {"accessibility requirement", func(r *Replay) { r.Roster[0].Accessibility = "plain-text email required" }}, {"individual assignment", func(r *Replay) { r.Roster[0].Assignment = "essential lab monitoring" }}}
	for _, c := range changes {
		t.Run(c.name, func(t *testing.T) {
			r, _ := newReplay("barnwell", nil)
			r.approve()
			binding := r.Approval.Binding
			archive := digest(r.ApprovedPackages)
			firstEvent := digest(r.History[0])
			originalNotice := r.Roster[0].Original
			c.mutate(r)
			if digest(packageFor(r)) == binding {
				t.Fatal("recipient profile excluded from exact approval")
			}
			if _, err := r.simulate(); err == nil || len(r.Intents) != 0 {
				t.Fatal("changed profile used old approval")
			}
			if digest(r.ApprovedPackages) != archive || digest(r.History[0]) != firstEvent || r.Roster[0].Original != originalNotice {
				t.Fatal("original approval or notice history changed")
			}
		})
	}
}
func TestOperatingPostureChangeInvalidatesApprovalAndPreservesOriginal(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	r.approve()
	binding := r.Approval.Binding
	oldPackage := digest(r.ApprovedPackages[0])
	firstEvent := digest(r.History[0])
	if err := r.selectPosture("closure"); err != nil {
		t.Fatal(err)
	}
	if r.OperatingPosture != "closure" || r.Approval != nil || packageFor(r).Parameters["operatingPosture"] != "closure" {
		t.Fatal("posture not bound/reviewed")
	}
	if _, err := r.simulate(); err == nil {
		t.Fatal("posture change sent without approval")
	}
	if digest(r.ApprovedPackages[0]) != oldPackage || digest(r.History[0]) != firstEvent {
		t.Fatal("history changed")
	}
	if err := r.approve(); err != nil {
		t.Fatal(err)
	}
	if r.Approval.Binding == binding || r.Approval.Package.Parameters["operatingPosture"] != "closure" || len(r.ApprovedPackages) != 2 {
		t.Fatal("new posture approval not exact")
	}
	before := digest(r)
	for _, posture := range []string{"normal", "invented-posture"} {
		if err := r.selectPosture(posture); err == nil || digest(r) != before {
			t.Fatal("normal or unsupported posture changed approved state")
		}
	}
}
func TestHTTPPostureApprovalPersistsExactSelectedParameter(t *testing.T) {
	app := qaBriefingApp(t, nil)
	r := app.replays["barnwell"]
	messages := map[string]string{}
	for _, m := range r.Messages {
		messages[m.Group] = m.Text
	}
	raw, _ := json.Marshal(map[string]any{"action": "approve", "revision": digest(packageFor(r)), "operatingPosture": "closure", "messages": messages})
	path := "/api/replay/action?facility=barnwell"
	app.auth = qaAuthorization()
	request := qaSignedRequest(qaPrincipal("POST", path, raw), "POST", path, raw)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	r = app.replays["barnwell"]
	if r.Approval == nil || r.Approval.Package.Parameters["operatingPosture"] != "closure" || r.Approval.ActorID != "google:123" {
		t.Fatal("posture or manager lost", r.Approval)
	}
}
