package weatherdesk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Principal is issued by the authenticated frontend, never supplied by the model.
type Principal struct {
	Sub        string   `json:"sub"`
	Name       string   `json:"name"`
	Login      string   `json:"login"`
	Role       string   `json:"role"`
	Facilities []string `json:"facilities"`
	Exp        int64    `json:"exp"`
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	BodySHA256 string   `json:"body_sha256"`
	Nonce      string   `json:"nonce"`
}
type principalKey struct{}
type authorization struct {
	required bool
	secret   []byte
	origin   string
	mu       sync.Mutex
	nonces   map[string]int64
}

func newAuthorization() (*authorization, error) {
	if addr := os.Getenv("WEATHER_DESK_HTTP_ADDR"); addr != "" && addr != "127.0.0.1:8087" && addr != "localhost:8087" && os.Getenv("AUTH_REQUIRED") != "true" {
		return nil, errors.New("public API binding requires AUTH_REQUIRED=true")
	}
	a := &authorization{required: os.Getenv("AUTH_REQUIRED") == "true", secret: []byte(os.Getenv("WEATHER_DESK_GATEWAY_SECRET")), origin: os.Getenv("AUTH_URL"), nonces: map[string]int64{}}
	if a.required && len(a.secret) < 32 {
		return nil, errors.New("AUTH_REQUIRED needs a gateway secret of at least 32 bytes")
	}
	return a, nil
}
func (a *authorization) verify(r *http.Request, facility string) (Principal, error) {
	if !a.required {
		return Principal{Sub: "demo:fixture-manager", Name: "Demo facility manager", Role: "fixture-manager", Facilities: []string{facility}}, nil
	}
	encoded := r.Header.Get("X-Weather-Principal")
	signature, err := hex.DecodeString(r.Header.Get("X-Weather-Signature"))
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(encoded))
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return Principal{}, errors.New("authentication required")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) > 4096 {
		return Principal{}, errors.New("invalid principal")
	}
	var p Principal
	if json.Unmarshal(raw, &p) != nil {
		return p, errors.New("invalid principal")
	}
	now := time.Now().Unix()
	if p.Sub == "" || p.Exp <= now || p.Exp > now+65 || p.Method != r.Method || p.Path != r.URL.RequestURI() || len(p.Nonce) < 16 {
		return p, errors.New("expired or mismatched principal")
	}
	allowed := false
	for _, id := range p.Facilities {
		if id == facility {
			allowed = true
		}
	}
	if !allowed {
		return p, errors.New("facility access denied")
	}
	if p.Role != "manager" && p.Role != "viewer" {
		return p, errors.New("invalid role")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32769))
	if err != nil || len(body) > 32768 {
		return p, errors.New("invalid request body")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	hash := sha256.Sum256(body)
	if p.BodySHA256 != hex.EncodeToString(hash[:]) {
		return p, errors.New("request body mismatch")
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if p.Role != "manager" {
			return p, errors.New("manager approval authority required")
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		for nonce, expiry := range a.nonces {
			if expiry <= now {
				delete(a.nonces, nonce)
			}
		}
		if _, seen := a.nonces[p.Nonce]; seen {
			return p, errors.New("request was already used")
		}
		a.nonces[p.Nonce] = p.Exp
	}
	return p, nil
}
func requestPrincipal(r *http.Request) Principal {
	p, _ := r.Context().Value(principalKey{}).(Principal)
	return p
}
func withPrincipal(r *http.Request, p Principal) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), principalKey{}, p))
}
func policyInputFor(r *Replay, action string, p Principal) map[string]any {
	input := policyInput(r, action)
	input["actor_role"] = p.Role
	input["actor_facility"] = ""
	for _, id := range p.Facilities {
		if id == r.Facility.ID {
			input["actor_facility"] = id
		}
	}
	return input
}
