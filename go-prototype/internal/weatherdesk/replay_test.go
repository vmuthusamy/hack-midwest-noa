package weatherdesk

import "testing"

func TestAllRecordedFacilitiesInsideTheirWarnings(t *testing.T) {
	for _, scenario := range []string{"barnwell", "florida", "california"} {
		r, err := newReplay(scenario, nil)
		if err != nil || r.Alert.Applicability != "applicable" {
			t.Fatalf("%s: %v, %+v", scenario, err, r)
		}
	}
}
func TestApprovalRecheckAndDeduplication(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	if _, err := r.simulate(); err == nil {
		t.Fatal("send without approval")
	}
	r.approve()
	n, err := r.simulate()
	if err != nil || n != 5 {
		t.Fatal(n, err)
	}
	n, err = r.simulate()
	if err != nil || n != 0 {
		t.Fatal(n, err)
	}
	for _, p := range r.Roster {
		if p.ID == "S01" && p.Original != "delivered" {
			t.Fatal("original notice overwritten")
		}
	}
	for _, i := range r.Intents {
		if i.State != "queued" || !i.Simulated || i.Recipient == "S06" {
			t.Fatal(i)
		}
	}
}
func TestMaterialChangesAndIrrelevantUpdate(t *testing.T) {
	for _, kind := range []string{"outside", "missing", "stale", "update", "cancel", "expired"} {
		t.Run(kind, func(t *testing.T) {
			r, _ := newReplay("barnwell", nil)
			r.approve()
			r.setCase(kind)
			if r.Approval != nil {
				t.Fatal("approval retained")
			}
			if _, err := r.simulate(); err == nil {
				t.Fatal("send accepted")
			}
		})
	}
	r, _ := newReplay("barnwell", nil)
	r.approve()
	binding := r.Approval.Binding
	r.setCase("irrelevant")
	if r.Approval == nil || r.Approval.Binding != binding {
		t.Fatal("irrelevant change invalidated approval")
	}
}
func TestExactPackageBinding(t *testing.T) {
	changes := []struct {
		name   string
		mutate func(*Replay)
	}{{"facility", func(r *Replay) { r.Facility.Latitude += 1 }}, {"message", func(r *Replay) { r.Messages[0].Text += " edited" }}, {"recipients", func(r *Replay) { r.Messages[0].Recipients = append(r.Messages[0].Recipients, "S06") }}, {"policy", func(r *Replay) { r.Policy = "v2" }}}
	for _, c := range changes {
		t.Run(c.name, func(t *testing.T) {
			r, _ := newReplay("barnwell", nil)
			r.approve()
			c.mutate(r)
			if _, err := r.simulate(); err == nil {
				t.Fatal("changed package sent")
			}
		})
	}
}
func TestMessageBuilderRequiresExactGroups(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	r.approve()
	if r.updateMessages(map[string]string{"onsite": "new", "arriving": " "}) == nil {
		t.Fatal("blank accepted")
	}
	r.updateMessages(map[string]string{"onsite": "new reviewed text", "arriving": "new arrival text"})
	if r.Approval != nil {
		t.Fatal("edit retained approval")
	}
}

func TestApprovalHistorySurvivesMessageChanges(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	r.approve()
	original := r.ApprovedPackages[0].Package.Messages[0].Text
	r.updateMessages(map[string]string{"onsite": "reviewed replacement", "arriving": "reviewed arrival replacement"})
	if r.Approval != nil || len(r.ApprovedPackages) != 1 || r.ApprovedPackages[0].Package.Messages[0].Text != original {
		t.Fatal("original approval package was altered")
	}
	r.approve()
	if len(r.ApprovedPackages) != 2 {
		t.Fatal("missing correction approval")
	}
}
