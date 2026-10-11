package weatherdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const incidentSource = "https://api.weather.gov/alerts/active"
const incidentLimit = 5000
const stateCodes = "AL AK AZ AR CA CO CT DE DC FL GA HI ID IL IN IA KS KY LA ME MD MA MI MN MS MO MT NE NV NH NJ NM NY NC ND OH OK OR PA RI SC SD TN TX UT VT VA WA WV WI WY AS GU MP PR VI"

type incidentEvent struct {
	ObservedAt string   `json:"observedAt"`
	Kind       string   `json:"kind"`
	ID         string   `json:"id"`
	Event      string   `json:"event"`
	Area       string   `json:"area"`
	Issued     string   `json:"issued"`
	Expires    string   `json:"expires"`
	Regions    []string `json:"regions"`
}
type incidentState struct {
	StartedAt   string          `json:"startedAt"`
	LastSuccess string          `json:"lastSuccessfulRefresh"`
	Feed        alertFeed       `json:"feed"`
	History     []incidentEvent `json:"history"`
	Pruned      int             `json:"prunedEvents"`
}
type incidentMonitor struct {
	mu                                                sync.RWMutex
	state                                             incidentState
	path, etag, modified, lastError, persistenceError string
	nextPoll                                          time.Time
	attempts                                          uint64
	ctx                                               context.Context
	cancel                                            context.CancelFunc
	zones                                             map[string][]string
	zoneChecked                                       map[string]time.Time
}

func newIncidentMonitor(root string) (*incidentMonitor, error) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &incidentMonitor{path: filepath.Join(root, ".cache", "incidents.json"), ctx: ctx, cancel: cancel, zones: map[string][]string{}, zoneChecked: map[string]time.Time{}}
	raw, err := os.ReadFile(m.path)
	if err == nil {
		err = json.Unmarshal(raw, &m.state)
	}
	if err != nil && !os.IsNotExist(err) {
		cancel()
		return nil, fmt.Errorf("cannot restore incident history: %w", err)
	}
	if m.state.StartedAt == "" {
		m.state.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return m, nil
}

// One collector for the deployment, independent of browsers, models and evals.
// If this service is replicated, elect one collector before adding replicas.
func (m *incidentMonitor) run() {
	delay := time.Minute
	for {
		started := time.Now()
		err := m.poll()
		m.mu.Lock()
		if err != nil {
			m.lastError = "NWS refresh failed; showing the last successful snapshot"
			delay = min(5*time.Minute, delay*2)
		} else {
			m.lastError = ""
			delay = time.Minute
		}
		m.nextPoll = started.Add(delay)
		next := time.Until(m.nextPoll)
		m.mu.Unlock()
		timer := time.NewTimer(max(time.Second, next))
		select {
		case <-m.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (m *incidentMonitor) poll() error {
	ctx, cancel := context.WithTimeout(m.ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, incidentSource, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WeatherDesk-HackMidwest-personal-experiment")
	req.Header.Set("Accept", "application/geo+json")
	m.mu.Lock()
	m.attempts++
	etag, modified := m.etag, m.modified
	m.mu.Unlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if modified != "" {
		req.Header.Set("If-Modified-Since", modified)
	}
	response, err := noaaClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var feed alertFeed
	if response.StatusCode == http.StatusOK {
		if err = json.NewDecoder(io.LimitReader(response.Body, 12<<20)).Decode(&feed); err != nil {
			return err
		}
		if feed.Features == nil {
			return fmt.Errorf("missing alert collection")
		}
	} else if response.StatusCode != http.StatusNotModified {
		return fmt.Errorf("NWS status %d", response.StatusCode)
	}
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	if response.StatusCode == http.StatusOK {
		previous := map[string]Feature{}
		for _, f := range m.state.Feed.Features {
			previous[featureID(f)] = f
		}
		initial := m.state.LastSuccess == ""
		for _, f := range feed.Features {
			id := featureID(f)
			old, found := previous[id]
			kind := ""
			if !found {
				kind = "First observed"
				if initial {
					kind = "Initial snapshot"
				}
			} else if digest(old) != digest(f) {
				kind = "Updated"
			}
			if kind != "" && f.Properties.Status == "Actual" {
				m.appendEvent(f, kind, now)
			}
			delete(previous, id)
		}
		for _, f := range previous {
			if f.Properties.Status == "Actual" {
				m.appendEvent(f, "Left active feed", now)
			}
		}
		m.state.Feed = feed
		m.etag = response.Header.Get("ETag")
		m.modified = response.Header.Get("Last-Modified")
	} else if m.state.LastSuccess == "" {
		return fmt.Errorf("304 without a snapshot")
	}
	m.state.LastSuccess = now.Format(time.RFC3339)
	cutoff := now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	keep := 0
	for keep < len(m.state.History) && m.state.History[keep].ObservedAt < cutoff {
		keep++
	}
	keep = max(keep, len(m.state.History)-incidentLimit)
	m.state.Pruned += keep
	m.state.History = append([]incidentEvent{}, m.state.History[keep:]...)
	if err = m.saveLocked(); err != nil {
		m.persistenceError = "History could not be saved; recent changes may be lost on restart"
	} else {
		m.persistenceError = ""
	}
	return nil
}
func featureID(f Feature) string {
	if f.ID != "" {
		return f.ID
	}
	return f.Properties.ID
}
func featureRegions(f Feature) []string {
	found := map[string]bool{}
	for _, code := range f.Properties.Geocode.UGC {
		if len(code) >= 2 {
			found[code[:2]] = true
		}
	}
	for _, zone := range f.Properties.Zones {
		code := zone[strings.LastIndex(zone, "/")+1:]
		if len(code) >= 2 {
			found[code[:2]] = true
		}
	}
	regions := []string{}
	for _, code := range strings.Fields(stateCodes) {
		if found[code] {
			regions = append(regions, code)
		}
	}
	if len(regions) == 0 {
		regions = append(regions, "OTHER")
	}
	return regions
}
func hasRegion(regions []string, region string) bool {
	if region == "" || region == "ALL" {
		return true
	}
	for _, r := range regions {
		if r == region {
			return true
		}
	}
	return false
}
func (m *incidentMonitor) appendEvent(f Feature, kind string, now time.Time) {
	p := f.Properties
	if p.MessageType == "Cancel" && kind != "Left active feed" {
		kind = "NWS cancellation observed"
	}
	if p.MessageType == "Update" && kind == "First observed" {
		kind = "NWS update observed"
	}
	m.state.History = append(m.state.History, incidentEvent{now.Format(time.RFC3339), kind, featureID(f), p.Event, p.Area, p.Sent, p.Expires, featureRegions(f)})
}
func (m *incidentMonitor) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(m.state)
	if err != nil {
		return err
	}
	if err = os.WriteFile(m.path+".tmp", raw, 0600); err != nil {
		return err
	}
	return os.Rename(m.path+".tmp", m.path)
}
func (m *incidentMonitor) feedForDecision() (alertFeed, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	checked, err := time.Parse(time.RFC3339, m.state.LastSuccess)
	if err != nil || time.Since(checked) > 90*time.Second || m.lastError != "" {
		return alertFeed{}, "", fmt.Errorf("Live monitoring is unavailable or overdue. Wait for a successful refresh before reviewing a new decision")
	}
	return m.state.Feed, m.state.LastSuccess, nil
}

// Zone lookups are bounded to configured facilities, refreshed every six hours.
func (m *incidentMonitor) refreshZones(facilities []Facility) {
	for {
		for _, f := range facilities {
			var point struct {
				Properties struct {
					ForecastZone    string `json:"forecastZone"`
					County          string `json:"county"`
					FireWeatherZone string `json:"fireWeatherZone"`
				} `json:"properties"`
			}
			if err := getNOAA(m.ctx, fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", f.Latitude, f.Longitude), &point); err == nil {
				zones := []string{}
				for _, z := range []string{point.Properties.ForecastZone, point.Properties.County, point.Properties.FireWeatherZone} {
					if nwsZoneKind(z) != "" {
						zones = append(zones, z)
					}
				}
				m.mu.Lock()
				m.zones[f.ID] = zones
				m.zoneChecked[f.ID] = time.Now()
				m.mu.Unlock()
			}
			if m.ctx.Err() != nil {
				return
			}
		}
		timer := time.NewTimer(6 * time.Hour)
		select {
		case <-m.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (m *incidentMonitor) view(region string, f Facility) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().UTC()
	checked, _ := time.Parse(time.RFC3339, m.state.LastSuccess)
	health := "current"
	if m.state.LastSuccess == "" {
		health = "starting"
	} else if now.Sub(checked) > 90*time.Second || m.lastError != "" {
		health = "overdue"
	}
	zones := m.zones[f.ID]
	if time.Since(m.zoneChecked[f.ID]) > 24*time.Hour {
		zones = nil
	}
	counts := map[string]int{}
	alerts := []Alert{}
	for _, feature := range m.state.Feed.Features {
		if feature.Properties.Status != "Actual" {
			continue
		}
		alert := normalize(feature, f, zones, now)
		if alert.Lifecycle == "expired" || alert.Lifecycle == "cancelled" {
			continue
		}
		regions := featureRegions(feature)
		for _, r := range regions {
			counts[r]++
		}
		if hasRegion(regions, region) {
			// The monitor serves compact summaries; the manager case retains full source evidence.
			alert.Description = ""
			alert.Geometry = nil
			alert.References = nil
			alerts = append(alerts, alert)
		}
	}
	sort.SliceStable(alerts, func(i, j int) bool {
		if alertRank(alerts[i]) != alertRank(alerts[j]) {
			return alertRank(alerts[i]) < alertRank(alerts[j])
		}
		return alerts[i].Issued > alerts[j].Issued
	})
	history := []incidentEvent{}
	total := 0
	for i := len(m.state.History) - 1; i >= 0; i-- {
		e := m.state.History[i]
		if hasRegion(e.Regions, region) {
			total++
			if len(history) < 100 {
				history = append(history, e)
			}
		}
	}
	return map[string]any{"mode": "LIVE", "health": health, "lastSuccessfulRefresh": m.state.LastSuccess, "feedUpdated": m.state.Feed.Updated, "nextPollAt": m.nextPoll, "pollSeconds": 60, "startedAt": m.state.StartedAt, "source": incidentSource, "alerts": alerts, "regionCounts": counts, "history": history, "historyTotal": total, "prunedEvents": m.state.Pruned, "error": m.lastError, "persistenceError": m.persistenceError, "pollAttemptsSinceRestart": m.attempts, "facility": f, "facilityZonesKnown": len(zones) > 0}
}
