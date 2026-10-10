package weatherdesk

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGeographyBoundaries(t *testing.T) {
	polygon := json.RawMessage(`{"type":"Polygon","coordinates":[[[0,0],[2,0],[2,2],[0,2],[0,0]],[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5],[0.5,0.5]]]}`)
	cases := []struct {
		name     string
		lon, lat float64
		want     string
	}{{"inside", 0.2, 0.2, "applicable"}, {"outside", 3, 3, "outside"}, {"hole", 1, 1, "outside"}, {"boundary", 0, 1, "applicable"}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := geography(polygon, Facility{Longitude: c.lon, Latitude: c.lat}, nil, nil)
			if got != c.want {
				t.Fatalf("%s != %s", got, c.want)
			}
		})
	}
}
func TestZoneMatchingAndPolygonPrecedence(t *testing.T) {
	zones := []string{"https://api.weather.gov/zones/forecast/FLZ028", "https://api.weather.gov/zones/county/FLC123"}
	got, _ := geography(nil, Facility{}, zones[:1], zones)
	if got != "applicable" {
		t.Fatal(got)
	}
	got, _ = geography(nil, Facility{}, zones[:1], nil)
	if got != "unknown" {
		t.Fatal(got)
	}
	got, _ = geography(json.RawMessage(`{"type":"Polygon","coordinates":[[[1,1],[2,1],[2,2],[1,2],[1,1]]]}`), Facility{}, zones, zones)
	if got != "outside" {
		t.Fatal(got)
	}
	got, _ = geography(json.RawMessage(`{"type":"Polygon","coordinates":[]}`), Facility{}, zones, zones)
	if got != "unknown" {
		t.Fatal(got)
	}
}
func TestAlertTimeAndLifecycle(t *testing.T) {
	var recording Recording
	b, _ := assets.ReadFile("fixtures/barnwell.json")
	json.Unmarshal(b, &recording)
	now, _ := time.Parse(time.RFC3339, "2026-10-10T21:07:32Z")
	cases := []struct {
		name   string
		mutate func(*Feature)
		want   string
	}{{"active", func(f *Feature) {}, "applicable"}, {"outside", func(f *Feature) {
		f.Geometry = json.RawMessage(`{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}`)
	}, "not-applicable"}, {"missing", func(f *Feature) { f.Geometry = nil; f.Properties.Zones = nil }, "unknown"}, {"missing time", func(f *Feature) { f.Properties.Expires = "" }, "unknown"}, {"cancelled", func(f *Feature) { f.Properties.MessageType = "Cancel" }, "not-applicable"}, {"expired", func(f *Feature) { f.Properties.Expires = "2026-10-10T21:07:32Z" }, "not-applicable"}, {"test alert", func(f *Feature) { f.Properties.Status = "Test" }, "not-applicable"}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := recording.Feature
			c.mutate(&f)
			a := normalize(f, recording.Facility, nil, now)
			if a.Applicability != c.want {
				t.Fatalf("got %s", a.Applicability)
			}
		})
	}
}
