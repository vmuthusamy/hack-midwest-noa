package weatherdesk

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPApprovalGateAndReplayRevision(t *testing.T) {
	app, err := New(t.TempDir(), "127.0.0.1:8089")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	request := func(body map[string]any, origin string) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/api/replay/action?facility=barnwell", bytes.NewReader(raw))
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	replay := app.replays["barnwell"]
	revision := digest(packageFor(replay))
	w := request(map[string]any{"action": "approve", "revision": revision}, "https://unrelated.example")
	if w.Code != http.StatusForbidden {
		t.Fatal(w.Code)
	}
	w = request(map[string]any{"action": "approve", "revision": "old-revision"}, "http://127.0.0.1:8088")
	if w.Code != http.StatusConflict {
		t.Fatal(w.Code)
	}
	w = request(map[string]any{"action": "simulate", "revision": revision}, "http://127.0.0.1:8088")
	if w.Code != http.StatusConflict {
		t.Fatal(w.Code)
	}
	w = request(map[string]any{"action": "approve", "revision": revision}, "http://127.0.0.1:8088")
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	replay = app.replays["barnwell"]
	w = request(map[string]any{"action": "simulate", "revision": revision, "binding": replay.Approval.Binding}, "http://127.0.0.1:8088")
	if w.Code != http.StatusOK || len(app.replays["barnwell"].Intents) != 5 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(map[string]any{"action": "simulate", "revision": revision, "binding": replay.Approval.Binding}, "http://127.0.0.1:8088")
	if w.Code != http.StatusOK || len(app.replays["barnwell"].Intents) != 5 {
		t.Fatal("duplicate", w.Code, w.Body.String())
	}
}

func TestFailedPersistenceDoesNotPublishApproval(t *testing.T) {
	root := t.TempDir()
	app, err := New(root, "127.0.0.1:8089")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	os.WriteFile(filepath.Join(root, ".cache"), []byte("not a directory"), 0600)
	before := digest(app.replays["barnwell"])
	revision := digest(packageFor(app.replays["barnwell"]))
	raw, _ := json.Marshal(map[string]string{"action": "approve", "revision": revision})
	req := httptest.NewRequest("POST", "/api/replay/action?facility=barnwell", bytes.NewReader(raw))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 500 || digest(app.replays["barnwell"]) != before {
		t.Fatal("failed save published state", w.Code)
	}
}
