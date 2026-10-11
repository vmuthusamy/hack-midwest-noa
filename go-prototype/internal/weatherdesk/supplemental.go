package weatherdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RiverSignal is regional context, not evidence that a facility or access road floods.
// Values retain each provider's gauge reference. Missing values are JSON null, never zero.
type RiverSignal struct {
	Kind                  string          `json:"kind"`
	Provider              string          `json:"provider"`
	GaugeID               string          `json:"gaugeId"`
	USGSID                string          `json:"usgsId"`
	Name                  string          `json:"name"`
	Source                string          `json:"source"`
	MetadataSource        string          `json:"metadataSource"`
	ObservedAt            string          `json:"observedAt"`
	AttemptedAt           string          `json:"attemptedAt"`
	FetchedAt             string          `json:"fetchedAt"`
	Value                 *float64        `json:"value"`
	Unit                  string          `json:"unit"`
	ParameterCode         string          `json:"parameterCode"`
	ApprovalStatus        string          `json:"approvalStatus"`
	Latitude              *float64        `json:"latitude"`
	Longitude             *float64        `json:"longitude"`
	ObservationAgeSeconds *int64          `json:"observationAgeSeconds"`
	State                 string          `json:"state"`
	ReportedCategory      string          `json:"reportedCategory"`
	Thresholds            RiverThresholds `json:"thresholds"`
	MinorStageDifference  *float64        `json:"minorStageDifference"`
	Relevance             string          `json:"relevance"`
	FacilityImpact        string          `json:"facilityImpact"`
	RelevanceExplanation  string          `json:"relevanceExplanation"`
	Gaps                  []string        `json:"gaps"`
}

type RiverThresholds struct {
	Source   string   `json:"source"`
	Unit     string   `json:"unit"`
	Action   *float64 `json:"action"`
	Minor    *float64 `json:"minor"`
	Moderate *float64 `json:"moderate"`
	Major    *float64 `json:"major"`
}

type riverConfig struct {
	lid, site, name, observationSource string
	usgs                               bool
}

func riverFor(f Facility) (riverConfig, bool) {
	switch f.ID {
	case "SC-DEMO-01":
		return riverConfig{"SRBS1", "02175200", "Salkehatchie River near Barnwell", "https://api.waterdata.usgs.gov/ogcapi/v1/collections/latest-continuous/items?monitoring_location_id=USGS-02175200&parameter_code=00065&f=json&limit=20", true}, true
	case "FL-DEMO-01":
		// NOAA's same-response stage and categories avoid mixing the stale USGS
		// record with a different stage reference at this location.
		return riverConfig{"STEF1", "02324000", "Steinhatchee River near Cross City", "https://api.water.noaa.gov/nwps/v1/gauges/stef1", false}, true
	default:
		return riverConfig{}, false
	}
}

type nwpsRiver struct {
	LID       string   `json:"lid"`
	USGSID    string   `json:"usgsId"`
	Name      string   `json:"name"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Status    struct {
		Observed struct {
			Primary       *float64 `json:"primary"`
			PrimaryUnit   string   `json:"primaryUnit"`
			FloodCategory string   `json:"floodCategory"`
			ValidTime     string   `json:"validTime"`
		} `json:"observed"`
	} `json:"status"`
	Flood struct {
		StageUnits string `json:"stageUnits"`
		Categories struct {
			Action struct {
				Stage *float64 `json:"stage"`
			} `json:"action"`
			Minor struct {
				Stage *float64 `json:"stage"`
			} `json:"minor"`
			Moderate struct {
				Stage *float64 `json:"stage"`
			} `json:"moderate"`
			Major struct {
				Stage *float64 `json:"stage"`
			} `json:"major"`
		} `json:"categories"`
	} `json:"flood"`
}

type usgsRiver struct {
	Features []struct {
		Properties struct {
			Site      string  `json:"monitoring_location_id"`
			Parameter string  `json:"parameter_code"`
			Time      string  `json:"time"`
			Value     *string `json:"value"`
			Unit      string  `json:"unit_of_measure"`
			Approval  string  `json:"approval_status"`
		} `json:"properties"`
		Geometry struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func riverNumber(v *float64) *float64 {
	if v == nil || math.IsNaN(*v) || math.IsInf(*v, 0) || *v == -999 || *v == -9999 {
		return nil
	}
	copy := *v
	return &copy
}

func riverObservation(s *RiverSignal, value *float64, unit, observed string, now time.Time) {
	s.Value, s.ObservationAgeSeconds, s.State = nil, nil, "unknown"
	s.Unit, s.ObservedAt = unit, observed
	parsed, err := time.Parse(time.RFC3339, observed)
	if err != nil || parsed.After(now) {
		s.Gaps = append(s.Gaps, "River observation time is missing, invalid or in the future.")
		return
	}
	age := int64(now.Sub(parsed).Seconds())
	s.ObservationAgeSeconds = &age
	if unit != "ft" || riverNumber(value) == nil {
		s.Gaps = append(s.Gaps, "River stage or compatible feet units are unavailable.")
		return
	}
	s.Value = riverNumber(value)
	s.State = "available"
}

// FetchRiverSignals reads the public USGS/NWPS sources configured for the demo facility.
// Observation age is reported, not hidden behind an invented hazard freshness threshold.
// The caller's policy decides whether data age is usable for its operating decision.
func FetchRiverSignals(ctx context.Context, f Facility) ([]RiverSignal, error) {
	return fetchRiverSignals(ctx, f, time.Now().UTC(), getRiverJSON)
}

func fetchRiverSignals(ctx context.Context, f Facility, now time.Time, get func(context.Context, string, any) error) ([]RiverSignal, error) {
	cfg, ok := riverFor(f)
	if !ok {
		return []RiverSignal{}, nil
	}
	metadata := "https://api.water.noaa.gov/nwps/v1/gauges/" + strings.ToLower(cfg.lid)
	s := RiverSignal{Kind: "river-stage", Provider: "NOAA NWPS", GaugeID: cfg.lid, USGSID: cfg.site, Name: cfg.name, Source: cfg.observationSource, MetadataSource: metadata, AttemptedAt: now.Format(time.RFC3339), State: "unknown", Relevance: "regional-context", FacilityImpact: "unknown", RelevanceExplanation: "Configured regional river context for this fictional facility. No surveyed inundation or access-road dependency is supplied.", Gaps: []string{"Facility or access-road flood impact is unknown; proximity alone is not applicability."}}
	if cfg.usgs {
		s.Provider = "USGS"
		s.ParameterCode = "00065"
	}
	var station nwpsRiver
	metadataErr := get(ctx, metadata, &station)
	if metadataErr == nil && (station.LID != cfg.lid || station.USGSID != cfg.site) {
		metadataErr = fmt.Errorf("river metadata gauge identity mismatch")
	}
	if metadataErr != nil {
		s.Gaps = append(s.Gaps, "Published river metadata unavailable; thresholds remain unknown.")
	} else {
		s.Thresholds = RiverThresholds{Source: metadata, Unit: station.Flood.StageUnits, Action: riverNumber(station.Flood.Categories.Action.Stage), Minor: riverNumber(station.Flood.Categories.Minor.Stage), Moderate: riverNumber(station.Flood.Categories.Moderate.Stage), Major: riverNumber(station.Flood.Categories.Major.Stage)}
		if !cfg.usgs {
			s.FetchedAt = now.Format(time.RFC3339)
			if station.Latitude != nil && station.Longitude != nil && math.Abs(*station.Latitude) <= 90 && math.Abs(*station.Longitude) <= 180 {
				s.Latitude, s.Longitude = riverNumber(station.Latitude), riverNumber(station.Longitude)
			}
			s.ReportedCategory = station.Status.Observed.FloodCategory
			riverObservation(&s, station.Status.Observed.Primary, station.Status.Observed.PrimaryUnit, station.Status.Observed.ValidTime, now)
		}
	}
	var observationErr error
	if cfg.usgs {
		var collection usgsRiver
		observationErr = get(ctx, cfg.observationSource, &collection)
		if observationErr == nil {
			s.FetchedAt = now.Format(time.RFC3339)
			latest := time.Time{}
			selected := -1
			for i, feature := range collection.Features {
				p := feature.Properties
				if p.Site != "USGS-"+cfg.site || p.Parameter != "00065" {
					continue
				}
				observed, err := time.Parse(time.RFC3339, p.Time)
				if err != nil || !observed.After(latest) {
					continue
				}
				latest = observed
				selected = i
			}
			if selected >= 0 {
				feature := collection.Features[selected]
				p := feature.Properties
				s.ApprovalStatus = p.Approval
				var value *float64
				if p.Value != nil {
					if v, err := strconv.ParseFloat(*p.Value, 64); err == nil {
						value = &v
					}
				}
				riverObservation(&s, value, p.Unit, p.Time, now)
				coords := feature.Geometry.Coordinates
				if feature.Geometry.Type == "Point" && len(coords) == 2 && math.Abs(coords[0]) <= 180 && math.Abs(coords[1]) <= 90 {
					s.Longitude, s.Latitude = riverNumber(&coords[0]), riverNumber(&coords[1])
				}
			}
			if selected < 0 {
				s.Gaps = append(s.Gaps, "No matching USGS river-stage observation returned.")
			}
		} else {
			s.Gaps = append(s.Gaps, "USGS river refresh failed; current stage is unknown.")
		}
	}
	if s.Thresholds.Minor == nil {
		s.Gaps = append(s.Gaps, "No published minor-flood stage for this gauge; threshold comparison is unknown.")
	}
	// Comparing the USGS and NOAA stage references requires independent datum
	// verification. Only the NOAA same-response values are compared here.
	if !cfg.usgs && s.State == "available" && s.Thresholds.Minor != nil && s.Unit == s.Thresholds.Unit {
		difference := *s.Value - *s.Thresholds.Minor
		s.MinorStageDifference = &difference
	}
	return []RiverSignal{s}, errors.Join(metadataErr, observationErr)
}

var riverClient = &http.Client{Timeout: 12 * time.Second}

func getRiverJSON(ctx context.Context, source string, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WeatherDesk-HackMidwest-public-data-experiment")
	req.Header.Set("Accept", "application/json")
	r, err := riverClient.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("river source returned HTTP %d", r.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(output)
}
