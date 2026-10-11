// Command posture-eval scores location matching and posture recommendations against
// real NWS alerts recorded during Hurricane Isaias (2026-10-11T02:20Z).
//
// Oracles are independent of the code under test:
//   - location: NWS's own /alerts/active?point= answer for Barnwell, captured at the same moment,
//     plus a separate point-in-polygon check for polygon alerts;
//   - posture: labels in posture-labels.json, written before this evaluator ran.
//
// Usage (from go-prototype): go run ./cmd/posture-eval ../fixtures/isaias-2026-10-10 [out.json]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	wd "github.com/vmuthusamy/hack-midwest-noa/go-prototype/internal/weatherdesk"
)

type collection struct {
	Features []wd.Feature `json:"features"`
}

type labels struct {
	Postures map[string]string `json:"postures"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func load(dir, pattern string, out any) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	must(err)
	if len(matches) == 0 {
		must(fmt.Errorf("no file matches %s", pattern))
	}
	b, err := os.ReadFile(matches[0])
	must(err)
	must(json.Unmarshal(b, out))
}

// Ray casting on the outer ring; independent of the app's geometry code.
func inPolygon(raw json.RawMessage, lon, lat float64) (bool, bool) {
	var g struct {
		Type        string          `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &g) != nil || g.Type != "Polygon" || len(g.Coordinates) == 0 {
		return false, false
	}
	ring, inside := g.Coordinates[0], false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		xi, yi, xj, yj := ring[i][0], ring[i][1], ring[j][0], ring[j][1]
		if (yi > lat) != (yj > lat) && lon < (xj-xi)*(lat-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside, true
}

func posture(p wd.PostureAssessment) string {
	if p.RecommendedPosture != nil {
		return *p.RecommendedPosture
	}
	if p.Status == "planning-review" {
		return "planning"
	}
	return "none"
}

type miss struct {
	Event    string `json:"event"`
	Area     string `json:"area"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
}

func main() {
	if len(os.Args) < 2 {
		must(fmt.Errorf("usage: posture-eval <fixture-dir> [out.json]"))
	}
	dir := os.Args[1]
	var all, point collection
	var pts struct {
		Properties struct{ ForecastZone, County, FireWeatherZone string } `json:"properties"`
	}
	var lab labels
	load(dir, "alerts-southeast-*.json", &all)
	load(dir, "alerts-point-barnwell-*.json", &point)
	load(dir, "points-barnwell-*.json", &pts)
	load(dir, "posture-labels.json", &lab)
	captured := time.Date(2026, 10, 11, 2, 20, 0, 0, time.UTC)

	// 1. Location vs NWS's own point query for Barnwell.
	barnwell := wd.Facility{ID: "SC-DEMO-01", Name: "Barnwell Field Office", Latitude: 33.28, Longitude: -81.35, Fictional: true}
	zones := []string{pts.Properties.ForecastZone, pts.Properties.County, pts.Properties.FireWeatherZone}
	oracle := map[string]bool{}
	for _, f := range point.Features {
		oracle[f.Properties.ID] = true
	}
	var tp, tn, fp, fn, unknown int
	for _, f := range all.Features {
		a := wd.NormalizeForEval(f, barnwell, zones, captured)
		want := oracle[f.Properties.ID]
		switch {
		case a.Geography == "unknown":
			unknown++
		case a.Geography == "applicable" && want:
			tp++
		case a.Geography == "applicable" && !want:
			fp++
		case a.Geography != "applicable" && want:
			fn++
		default:
			tn++
		}
	}

	// 2. Polygon alerts: app vs independent point-in-polygon, two facilities.
	facilities := []wd.Facility{barnwell, {ID: "SC-DEMO-02", Name: "Batesburg Field Office", Latitude: 33.91, Longitude: -81.54, Fictional: true}}
	var polyChecks, polyAgree int
	for _, f := range all.Features {
		for _, fac := range facilities {
			inside, ok := inPolygon(f.Geometry, fac.Longitude, fac.Latitude)
			if !ok {
				continue
			}
			a := wd.NormalizeForEval(f, fac, nil, captured)
			polyChecks++
			if (a.Geography == "applicable") == inside {
				polyAgree++
			}
		}
	}

	// 3. Posture vs labels, each alert treated as covering the facility and active.
	var scored, agree, missedWarnings, overEscalations, unlabeled, powerChecks, powerAgree int
	var misses []miss
	gapEvents := map[string]bool{}
	for _, f := range all.Features {
		want, ok := lab.Postures[f.Properties.Event]
		if !ok {
			unlabeled++
			continue
		}
		a := wd.NormalizeForEval(f, barnwell, zones, captured)
		a.Geography, a.Applicability, a.Lifecycle = "applicable", "applicable", "active"
		got := posture(wd.AssessPosture("REPLAY", barnwell, []wd.Alert{a}, true))
		scored++
		if got == want {
			agree++
		} else {
			misses = append(misses, miss{f.Properties.Event, f.Properties.Area, want, got})
			if (want == "telework" || want == "delay") && (got == "none" || got == "planning") {
				missedWarnings++
				gapEvents[f.Properties.Event] = true
			}
			if want == "none" && got != "none" {
				overEscalations++
			}
		}
		if want == "telework" || want == "delay" {
			e := wd.Evidence{Mode: "REPLAY", Facility: barnwell, Alerts: []wd.Alert{a}, PowerSignal: &wd.PowerSignal{Status: "yes"}}
			wd.ApplyPowerForEval(&e)
			powerChecks++
			if posture(e.PostureAssessment) == "closure" {
				powerAgree++
			}
		}
	}
	gaps := []string{}
	for e := range gapEvents {
		gaps = append(gaps, e)
	}
	sort.Strings(gaps)

	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(int(float64(a)/float64(b)*1000+0.5)) / 10
	}
	result := map[string]any{
		"dataset":  "68 real NWS alerts, SC/GA/NC/FL/AL, captured 2026-10-11T02:20Z during Hurricane Isaias",
		"location": map[string]any{"oracle": "NWS /alerts/active?point=33.28,-81.35 at capture", "alerts": len(all.Features), "correctInside": tp, "correctOutside": tn, "wrongInside": fp, "missedInside": fn, "unknown": unknown, "agreementPct": pct(tp+tn, len(all.Features))},
		"polygon":  map[string]any{"oracle": "independent point-in-polygon", "checks": polyChecks, "agree": polyAgree, "agreementPct": pct(polyAgree, polyChecks)},
		"posture":  map[string]any{"oracle": "posture-labels.json", "scored": scored, "agree": agree, "agreementPct": pct(agree, scored), "missedWarnings": missedWarnings, "overEscalations": overEscalations, "unlabeled": unlabeled, "policyGaps": gaps, "misses": misses},
		"power":    map[string]any{"rule": "warning + power out → closure", "checks": powerChecks, "agree": powerAgree},
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	if len(os.Args) > 2 {
		must(os.WriteFile(os.Args[2], append(out, '\n'), 0o644))
	}
}
