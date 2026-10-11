package weatherdesk

import "net/http"

// serveMetrics belongs behind the app's existing authenticated read boundary.
// It has no inference, operating approval or simulated sending authority.
func (a *App) serveMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		failure(w, http.StatusMethodNotAllowed, "Use GET for inference measurements")
		return
	}
	metrics, err := a.agent.Metrics(r.Context())
	if err != nil {
		failure(w, http.StatusServiceUnavailable, "Inference measurements unavailable; check the worker")
		return
	}
	respond(w, http.StatusOK, metrics)
}
