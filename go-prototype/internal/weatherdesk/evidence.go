package weatherdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Facility struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Fictional bool    `json:"fictional"`
}
type alertFeed struct {
	Features []Feature `json:"features"`
	Updated  string    `json:"updated"`
}

type Feature struct {
	ID         string          `json:"id"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties struct {
		Geocode struct {
			UGC []string `json:"UGC"`
		} `json:"geocode"`
		ID          string            `json:"id"`
		Event       string            `json:"event"`
		Headline    string            `json:"headline"`
		Description string            `json:"description"`
		Instruction string            `json:"instruction"`
		Area        string            `json:"areaDesc"`
		Issuer      string            `json:"senderName"`
		Sent        string            `json:"sent"`
		Effective   string            `json:"effective"`
		Expires     string            `json:"expires"`
		Onset       string            `json:"onset"`
		Ends        string            `json:"ends"`
		Status      string            `json:"status"`
		MessageType string            `json:"messageType"`
		Severity    string            `json:"severity"`
		Certainty   string            `json:"certainty"`
		Urgency     string            `json:"urgency"`
		Zones       []string          `json:"affectedZones"`
		References  []json.RawMessage `json:"references"`
	} `json:"properties"`
}
type Alert struct {
	ID            string            `json:"id"`
	Source        string            `json:"source"`
	Event         string            `json:"event"`
	Headline      string            `json:"headline"`
	Description   string            `json:"description"`
	Instruction   string            `json:"instruction"`
	Area          string            `json:"area"`
	Issuer        string            `json:"issuer"`
	Issued        string            `json:"issued"`
	Effective     string            `json:"effective"`
	Expires       string            `json:"expires"`
	Onset         string            `json:"onset"`
	Ends          string            `json:"ends"`
	Status        string            `json:"status"`
	MessageType   string            `json:"messageType"`
	Severity      string            `json:"severity"`
	Certainty     string            `json:"certainty"`
	Urgency       string            `json:"urgency"`
	Geography     string            `json:"geography"`
	Lifecycle     string            `json:"lifecycle"`
	Applicability string            `json:"applicability"`
	MatchMethod   string            `json:"matchMethod"`
	Geometry      json.RawMessage   `json:"geometry"`
	Zones         []string          `json:"affectedZones"`
	References    []json.RawMessage `json:"references"`
}
type Evidence struct {
	PowerSignal       *PowerSignal      `json:"powerSignal,omitempty"`
	NearbyAlerts      []Alert           `json:"nearbyAlerts,omitempty"`
	PostureAssessment PostureAssessment `json:"postureAssessment"`
	RiverSignals      []RiverSignal     `json:"riverSignals"`
	Mode              string            `json:"mode"`
	Facility          Facility          `json:"facility"`
	LastRefresh       string            `json:"lastSuccessfulRefresh"`
	FeedUpdated       string            `json:"feedUpdated"`
	Source            string            `json:"source"`
	Alerts            []Alert           `json:"alerts"`
	Gaps              []string          `json:"gaps"`
	Zones             []string          `json:"facilityZones"`
}

// Geometry takes precedence over county membership: a warning covers only its polygon.
func geography(raw json.RawMessage, f Facility, zones, facilityZones []string) (string, string) {
	if len(raw) == 0 || string(raw) == "null" {
		if len(zones) == 0 || len(facilityZones) == 0 {
			return "unknown", "location evidence missing"
		}
		// A nonmatching zone is outside only when the point lookup resolved
		// membership for that same zone type. Forecast and fire IDs can overlap.
		resolvedKinds := map[string]bool{}
		for _, fz := range facilityZones {
			if kind := nwsZoneKind(fz); kind != "" {
				resolvedKinds[kind] = true
			}
		}
		unresolved := false
		for _, z := range zones {
			kind := nwsZoneKind(z)
			if kind == "" || !resolvedKinds[kind] {
				unresolved = true
				continue
			}
			for _, fz := range facilityZones {
				if z == fz {
					return "applicable", "NWS point-to-zone membership"
				}
			}
		}
		if unresolved {
			return "unknown", "zone type or facility membership unresolved"
		}
		return "outside", "NWS point-to-zone membership"
	}
	var g struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if json.Unmarshal(raw, &g) != nil {
		return "unknown", "invalid geometry"
	}
	var polygons [][][][]float64
	switch g.Type {
	case "Polygon":
		var p [][][]float64
		if json.Unmarshal(g.Coordinates, &p) != nil {
			return "unknown", "invalid polygon"
		}
		polygons = append(polygons, p)
	case "MultiPolygon":
		if json.Unmarshal(g.Coordinates, &polygons) != nil {
			return "unknown", "invalid polygons"
		}
	default:
		return "unknown", "unsupported geometry"
	}
	if len(polygons) == 0 {
		return "unknown", "empty geometry"
	}
	for _, p := range polygons {
		if len(p) == 0 {
			return "unknown", "empty polygon"
		}
		for _, r := range p {
			if !validRing(r) {
				return "unknown", "invalid ring"
			}
		}
	}
	for _, p := range polygons {
		inside := contains(f.Longitude, f.Latitude, p[0])
		for _, hole := range p[1:] {
			if contains(f.Longitude, f.Latitude, hole) {
				inside = false
			}
		}
		if inside {
			return "applicable", "NWS warning polygon"
		}
	}
	return "outside", "NWS warning polygon"
}
func nwsZoneKind(zone string) string {
	for _, kind := range []string{"forecast", "county", "fire"} {
		prefix := "https://api.weather.gov/zones/" + kind + "/"
		if strings.HasPrefix(zone, prefix) && len(zone) > len(prefix) {
			return kind
		}
	}
	return ""
}

func validRing(r [][]float64) bool {
	if len(r) < 4 {
		return false
	}
	for _, p := range r {
		if len(p) != 2 || math.IsNaN(p[0]) || math.IsNaN(p[1]) || math.Abs(p[0]) > 180 || math.Abs(p[1]) > 90 {
			return false
		}
	}
	return r[0][0] == r[len(r)-1][0] && r[0][1] == r[len(r)-1][1]
}
func contains(x, y float64, r [][]float64) bool {
	inside := false
	for i := 0; i < len(r)-1; i++ {
		ax, ay, bx, by := r[i][0], r[i][1], r[i+1][0], r[i+1][1]
		cross := (x-ax)*(by-ay) - (y-ay)*(bx-ax)
		if math.Abs(cross) < 1e-10 && x >= math.Min(ax, bx) && x <= math.Max(ax, bx) && y >= math.Min(ay, by) && y <= math.Max(ay, by) {
			return true
		}
		if (ay > y) != (by > y) && x < (bx-ax)*(y-ay)/(by-ay)+ax {
			inside = !inside
		}
	}
	return inside
}
func normalize(f Feature, facility Facility, zones []string, now time.Time) Alert {
	p := f.Properties
	geo, method := geography(f.Geometry, facility, p.Zones, zones)
	start, e1 := time.Parse(time.RFC3339, p.Effective)
	end, e2 := time.Parse(time.RFC3339, p.Expires)
	life := "active"
	switch {
	case p.MessageType == "Cancel":
		life = "cancelled"
	case e1 != nil || e2 != nil || !end.After(start):
		life = "unknown"
	case now.Before(start):
		life = "not-yet-effective"
	case !now.Before(end):
		life = "expired"
	}
	applicable := "not-applicable"
	if p.Status == "Actual" && life == "active" {
		if geo == "applicable" {
			applicable = "applicable"
		} else if geo == "unknown" {
			applicable = "unknown"
		}
	}
	if life == "unknown" {
		applicable = "unknown"
	}
	id := p.ID
	if id == "" {
		id = f.ID
	}
	return Alert{ID: id, Source: f.ID, Event: p.Event, Headline: p.Headline, Description: p.Description, Instruction: p.Instruction, Area: p.Area, Issuer: p.Issuer, Issued: p.Sent, Effective: p.Effective, Expires: p.Expires, Onset: p.Onset, Ends: p.Ends, Status: p.Status, MessageType: p.MessageType, Severity: p.Severity, Certainty: p.Certainty, Urgency: p.Urgency, Geography: geo, Lifecycle: life, Applicability: applicable, MatchMethod: method, Geometry: f.Geometry, Zones: p.Zones, References: p.References}
}

var noaaClient = &http.Client{Timeout: 25 * time.Second}

func getNOAA(ctx context.Context, url string, output any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WeatherDesk-HackMidwest-personal-experiment")
	req.Header.Set("Accept", "application/geo+json")
	r, err := noaaClient.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("NWS returned %d", r.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(r.Body, 12<<20)).Decode(output)
}
func fetchEvidence(ctx context.Context, f Facility) (Evidence, error) {
	return fetchEvidenceUsing(ctx, f, nil)
}

func fetchEvidenceUsing(ctx context.Context, f Facility, monitor *incidentMonitor) (Evidence, error) {
	type pointResult struct {
		zones []string
		err   error
	}
	pointChan := make(chan pointResult, 1)
	go func() {
		if monitor != nil {
			monitor.mu.RLock()
			zones := append([]string{}, monitor.zones[f.ID]...)
			checked := monitor.zoneChecked[f.ID]
			monitor.mu.RUnlock()
			if time.Since(checked) > 24*time.Hour {
				zones = nil
			}
			pointChan <- pointResult{zones: zones}
			return
		}
		var p struct {
			Properties struct {
				ForecastZone    string `json:"forecastZone"`
				County          string `json:"county"`
				FireWeatherZone string `json:"fireWeatherZone"`
			} `json:"properties"`
		}
		err := getNOAA(ctx, fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", f.Latitude, f.Longitude), &p)
		var zones []string
		if err == nil {
			for _, membership := range []struct{ kind, zone string }{{"forecast", p.Properties.ForecastZone}, {"county", p.Properties.County}, {"fire", p.Properties.FireWeatherZone}} {
				if nwsZoneKind(membership.zone) == membership.kind {
					zones = append(zones, membership.zone)
				}
			}
		}
		pointChan <- pointResult{zones, err}
	}()
	var feed alertFeed
	source := "https://api.weather.gov/alerts/active"
	checkedAt := ""
	if monitor != nil {
		var err error
		feed, checkedAt, err = monitor.feedForDecision()
		if err != nil {
			return Evidence{}, err
		}
	} else if err := getNOAA(ctx, source, &feed); err != nil {
		return Evidence{}, err
	}
	if feed.Features == nil {
		return Evidence{}, fmt.Errorf("alert collection missing")
	}
	points := <-pointChan
	now := time.Now().UTC()
	if checkedAt == "" {
		checkedAt = now.Format(time.RFC3339)
	}
	e := Evidence{Mode: "LIVE", Facility: f, LastRefresh: checkedAt, FeedUpdated: feed.Updated, Source: source, Alerts: []Alert{}, Zones: points.zones, Gaps: []string{"No road, power, or employee commute data is connected.", "No prior operational approval is supplied to the live agent.", "An expired or cancelled warning does not authorize reopening."}}
	if points.err != nil || len(points.zones) == 0 {
		e.Gaps = append(e.Gaps, "Facility zone lookup failed; alerts without polygons have unknown applicability.")
	}
	if points.err == nil && len(points.zones) > 0 && len(points.zones) < 3 {
		e.Gaps = append(e.Gaps, "Some facility zone memberships are unavailable; alerts of unresolved types remain unknown.")
	}
	for _, feature := range feed.Features {
		e.Alerts = append(e.Alerts, normalize(feature, f, points.zones, now))
	}
	riverContext, cancelRiver := context.WithTimeout(ctx, 12*time.Second)
	defer cancelRiver()
	signals, riverError := FetchRiverSignals(riverContext, f)
	e.RiverSignals = signals
	if riverError != nil {
		e.Gaps = append(e.Gaps, "Supplementary river data is incomplete; see the gauge evidence gaps. Facility impact remains unknown.")
	}
	sort.SliceStable(e.Alerts, func(i, j int) bool { return alertRank(e.Alerts[i]) < alertRank(e.Alerts[j]) })
	e.PostureAssessment = AssessPosture(e.Mode, e.Facility, e.Alerts, true)
	return e, nil
}
func alertRank(a Alert) int {
	r := 300
	if a.Applicability == "applicable" {
		r = 0
	} else if a.Applicability == "unknown" {
		r = 100
	}
	if a.Lifecycle != "active" {
		r += 400
	}
	switch a.Event {
	case "Tornado Warning":
		return r
	case "Severe Thunderstorm Warning":
		return r + 1
	case "Flash Flood Warning":
		return r + 2
	case "Winter Storm Warning", "Flood Warning":
		return r + 3
	case "Flood Watch", "Winter Storm Watch":
		return r + 5
	}
	if a.Severity == "Extreme" {
		return r + 3
	}
	if a.Severity == "Severe" {
		return r + 4
	}
	return r + 6
}
