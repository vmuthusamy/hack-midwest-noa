package weatherdesk

import (
	"encoding/json"
	"time"
)

func recordedCaseEvidence(scenario string) (Evidence, error) {
	raw, err := assets.ReadFile("case-recordings/" + scenario + ".json")
	if err != nil {
		replay, err := newReplay(scenario, nil)
		if err != nil {
			return Evidence{}, err
		}
		return Evidence{Mode: "REPLAY", Facility: replay.Facility, LastRefresh: replay.CapturedAt, Source: replay.Alert.Source, Alerts: []Alert{replay.Alert}, RiverSignals: []RiverSignal{}, Gaps: []string{"Recorded NWS alert; synthetic facility, roster and operating policy. Replay uses the captured evidence clock."}, PostureAssessment: AssessPosture("REPLAY", replay.Facility, []Alert{replay.Alert}, true)}, nil
	}
	var record struct {
		CapturedAt  string     `json:"capturedAt"`
		Facility    Facility   `json:"facility"`
		Features    []Feature  `json:"features"`
		MapFeatures []Feature  `json:"mapFeatures"`
		Zones       []string   `json:"zones"`
		Source      string     `json:"source"`
		FeedUpdated string     `json:"feedUpdated"`
		River       *nwpsRiver `json:"river"`
	}
	if err = json.Unmarshal(raw, &record); err != nil {
		return Evidence{}, err
	}
	clock, err := time.Parse(time.RFC3339Nano, record.CapturedAt)
	if err != nil {
		return Evidence{}, err
	}
	e := Evidence{Mode: "REPLAY", Facility: record.Facility, LastRefresh: record.CapturedAt, Source: record.Source, FeedUpdated: record.FeedUpdated, Zones: record.Zones, Alerts: []Alert{}, RiverSignals: []RiverSignal{}, Gaps: []string{"Recorded NOAA source responses; synthetic facility, roster and operating policy. Replay clock is the capture time; these are not current conditions."}}
	for _, feature := range record.Features {
		e.Alerts = append(e.Alerts, normalize(feature, record.Facility, record.Zones, clock))
	}
	for _, feature := range record.MapFeatures {
		e.NearbyAlerts = append(e.NearbyAlerts, normalize(feature, record.Facility, record.Zones, clock))
	}
	if record.River != nil {
		station := record.River
		s := RiverSignal{Kind: "river-stage", Provider: "NOAA NWPS", GaugeID: station.LID, USGSID: station.USGSID, Name: station.Name, Source: "https://api.water.noaa.gov/nwps/v1/gauges/srbs1", MetadataSource: "https://api.water.noaa.gov/nwps/v1/gauges/srbs1", FetchedAt: record.CapturedAt, ReportedCategory: station.Status.Observed.FloodCategory, Relevance: "regional-context", FacilityImpact: "unknown", RelevanceExplanation: "Regional river context only; no facility or access-road dependency is supplied.", Gaps: []string{"Facility flood impact is unknown."}}
		s.Latitude, s.Longitude = riverNumber(station.Latitude), riverNumber(station.Longitude)
		s.Thresholds = RiverThresholds{Source: s.MetadataSource, Unit: station.Flood.StageUnits, Action: riverNumber(station.Flood.Categories.Action.Stage), Minor: riverNumber(station.Flood.Categories.Minor.Stage), Moderate: riverNumber(station.Flood.Categories.Moderate.Stage), Major: riverNumber(station.Flood.Categories.Major.Stage)}
		riverObservation(&s, station.Status.Observed.Primary, station.Status.Observed.PrimaryUnit, station.Status.Observed.ValidTime, clock)
		if s.Thresholds.Minor == nil {
			s.Gaps = append(s.Gaps, "No published flood-stage threshold at this gauge; comparison unavailable.")
		}
		e.RiverSignals = append(e.RiverSignals, s)
	}
	e.PostureAssessment = AssessPosture(e.Mode, e.Facility, e.Alerts, true)
	return e, nil
}
