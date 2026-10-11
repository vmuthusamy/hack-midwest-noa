package weatherdesk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFireZoneMembershipRequiresMatchingTypeEvidence(t *testing.T) {
	zones := []string{"https://api.weather.gov/zones/forecast/SCZ030", "https://api.weather.gov/zones/county/SCC011", "https://api.weather.gov/zones/fire/SCZ030"}
	cases := []struct {
		name             string
		alerts, facility []string
		want             string
	}{
		{"local", zones[2:], zones, "applicable"},
		{"single fire local", zones[2:], zones[2:], "applicable"},
		{"single fire outside", []string{"https://api.weather.gov/zones/fire/MIZ019"}, zones[2:], "outside"},
		{"single forecast local", zones[:1], zones[:1], "applicable"},
		{"Michigan", []string{"https://api.weather.gov/zones/fire/MIZ019"}, zones, "outside"},
		{"missing fire lookup", []string{"https://api.weather.gov/zones/fire/MIZ019"}, zones[:2], "unknown"},
		{"forecast cannot substitute for fire", zones[2:], zones[:2], "unknown"},
		{"unsupported type", []string{"https://api.weather.gov/zones/other/SCZ030"}, zones, "unknown"},
		{"mixed unresolved type", []string{"https://api.weather.gov/zones/fire/MIZ019", "https://api.weather.gov/zones/other/SCZ030"}, zones, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := geography(nil, Facility{}, c.alerts, c.facility)
			if got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

type zoneFixtureTransport func(*http.Request) (*http.Response, error)

func (f zoneFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLiveFireZoneOutsideDoesNotSuppressLocalWarning(t *testing.T) {
	var recording Recording
	b, _ := assets.ReadFile("fixtures/barnwell.json")
	if err := json.Unmarshal(b, &recording); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	local := recording.Feature
	local.Geometry = nil
	local.Properties.Zones = []string{"https://api.weather.gov/zones/forecast/SCZ030"}
	local.Properties.Sent = now.Add(-time.Minute).Format(time.RFC3339)
	local.Properties.Effective = local.Properties.Sent
	local.Properties.Expires = now.Add(time.Hour).Format(time.RFC3339)
	local.Properties.Status, local.Properties.MessageType = "Actual", "Alert"
	local.Properties.Severity, local.Properties.Urgency, local.Properties.Certainty = "Extreme", "Immediate", "Observed"
	elsewhere := local
	elsewhere.ID, elsewhere.Properties.ID = "https://api.weather.gov/alerts/urn:example:michigan-fire", "urn:example:michigan-fire"
	elsewhere.Properties.Event = "Red Flag Warning"
	elsewhere.Properties.Zones = []string{"https://api.weather.gov/zones/fire/MIZ019"}
	feed, err := json.Marshal(map[string]any{"features": []Feature{local, elsewhere}, "updated": now.Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	originalClient := noaaClient
	t.Cleanup(func() { noaaClient = originalClient })
	noaaClient = &http.Client{Transport: zoneFixtureTransport(func(r *http.Request) (*http.Response, error) {
		body := string(feed)
		if strings.HasPrefix(r.URL.Path, "/points/") {
			body = `{"properties":{"forecastZone":"https://api.weather.gov/zones/forecast/SCZ030","county":"https://api.weather.gov/zones/county/SCC011","fireWeatherZone":"https://api.weather.gov/zones/fire/SCZ030"}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	// An unmapped synthetic facility avoids unrelated gauge network calls.
	facility := recording.Facility
	facility.ID = "QA-ZONE-ONLY"
	evidence, err := fetchEvidence(context.Background(), facility)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Zones) != 3 {
		t.Fatalf("point lookup lost fire membership: %v", evidence.Zones)
	}
	if evidence.PostureAssessment.RecommendedPosture == nil || *evidence.PostureAssessment.RecommendedPosture != "delay" {
		t.Fatalf("unrelated fire warning suppressed local warning: %+v", evidence.PostureAssessment)
	}
	for _, a := range evidence.Alerts {
		if a.ID == elsewhere.Properties.ID && a.Geography != "outside" {
			t.Fatalf("Michigan alert applicability: %+v", a)
		}
	}
}

func TestPartialPointMembershipKeepsResolvedFireZone(t *testing.T) {
	original := noaaClient
	t.Cleanup(func() { noaaClient = original })
	for _, c := range []struct {
		name, points string
		count        int
		want         string
	}{
		{"fire only", `{"properties":{"fireWeatherZone":"https://api.weather.gov/zones/fire/SCZ030"}}`, 1, "outside"},
		{"forecast and fire", `{"properties":{"forecastZone":"https://api.weather.gov/zones/forecast/SCZ030","fireWeatherZone":"https://api.weather.gov/zones/fire/SCZ030"}}`, 2, "outside"},
		{"fire missing", `{"properties":{"forecastZone":"https://api.weather.gov/zones/forecast/SCZ030","county":"https://api.weather.gov/zones/county/SCC011"}}`, 2, "unknown"},
		{"mislabeled membership", `{"properties":{"fireWeatherZone":"https://api.weather.gov/zones/forecast/SCZ030"}}`, 0, "unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			now := time.Now().UTC()
			f := Feature{ID: "https://api.weather.gov/alerts/urn:qa:fire"}
			f.Properties.ID = "urn:qa:fire"
			f.Properties.Event = "Red Flag Warning"
			f.Properties.Zones = []string{"https://api.weather.gov/zones/fire/MIZ019"}
			f.Properties.Status = "Actual"
			f.Properties.MessageType = "Alert"
			f.Properties.Sent = now.Add(-time.Minute).Format(time.RFC3339)
			f.Properties.Effective = f.Properties.Sent
			f.Properties.Expires = now.Add(time.Hour).Format(time.RFC3339)
			feed, _ := json.Marshal(map[string]any{"features": []Feature{f}})
			noaaClient = &http.Client{Transport: zoneFixtureTransport(func(r *http.Request) (*http.Response, error) {
				body := string(feed)
				if strings.HasPrefix(r.URL.Path, "/points/") {
					body = c.points
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})}
			e, err := fetchEvidence(context.Background(), Facility{ID: "QA-PARTIAL", Latitude: 33.28, Longitude: -81.35})
			if err != nil {
				t.Fatal(err)
			}
			if len(e.Zones) != c.count || e.Alerts[0].Geography != c.want {
				t.Fatalf("partialmembership lost or inferred: zones=%v geo=%s", e.Zones, e.Alerts[0].Geography)
			}
		})
	}
}
