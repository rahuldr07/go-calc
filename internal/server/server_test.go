package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type sheet struct {
	Schema string `json:"schema"`
	Lines  []struct {
		Text   string `json:"text"`
		Status string `json:"status"`
		Error  string `json:"error"`
	} `json:"lines"`
}

func decodeSheet(t *testing.T, rec *httptest.ResponseRecorder) sheet {
	t.Helper()
	var s sheet
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return s
}

func TestPostSheet(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodPost, "/v1/sheet", `{"lines":["0.1 + 0.2","sqrt(2)"],"digits":12}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q", got)
	}
	s := decodeSheet(t, rec)
	if s.Schema != "go-calc.sheet/1" || len(s.Lines) != 2 {
		t.Fatalf("sheet = %+v", s)
	}
	if s.Lines[0].Text != "0.3" || s.Lines[0].Status != "exact" {
		t.Errorf("line 0 = %+v", s.Lines[0])
	}
}

func TestGetEval(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodGet, "/v1/eval?expr=sin(90)&angle=deg&digits=5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	s := decodeSheet(t, rec)
	if len(s.Lines) != 1 || s.Lines[0].Text != "1" {
		t.Errorf("sheet = %+v", s)
	}
}

func TestGetEvalMatchesPost(t *testing.T) {
	h := Handler(Config{})
	get := do(t, h, http.MethodGet, "/v1/eval?expr=1/3&digits=7", "")
	post := do(t, h, http.MethodPost, "/v1/sheet", `{"lines":["1/3"],"digits":7}`)
	if get.Body.String() != post.Body.String() {
		t.Errorf("get %s != post %s", get.Body, post.Body)
	}
}

func TestGetEvalNeedsExpr(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodGet, "/v1/eval", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d", rec.Code)
	}
}

func TestGetEvalRejectsBadDigits(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodGet, "/v1/eval?expr=1&digits=lots", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d", rec.Code)
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, body := range []string{`{"lines":`, `not json`, `[]`, `{"lines":"1+1"}`, ``} {
		rec := do(t, Handler(Config{}), http.MethodPost, "/v1/sheet", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status %d", body, rec.Code)
		}
	}
}

func TestWrongMethod(t *testing.T) {
	h := Handler(Config{})
	cases := []struct{ method, target, allow string }{
		{http.MethodGet, "/v1/sheet", http.MethodPost},
		{http.MethodPost, "/v1/eval?expr=1", http.MethodGet},
		{http.MethodPost, "/healthz", http.MethodGet},
	}
	for _, c := range cases {
		rec := do(t, h, c.method, c.target, "")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status %d", c.method, c.target, rec.Code)
		}
		if rec.Header().Get("Allow") != c.allow {
			t.Errorf("%s %s: Allow = %q", c.method, c.target, rec.Header().Get("Allow"))
		}
	}
}

func TestOversizedBody(t *testing.T) {
	body := `{"lines":["` + strings.Repeat("1", MaxBodyBytes) + `"]}`
	rec := do(t, Handler(Config{}), http.MethodPost, "/v1/sheet", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Errorf("status %d body %q", rec.Code, rec.Body)
	}
}

func TestRefusedLineIsStillOK(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodPost, "/v1/sheet", `{"lines":["1 +","2 + 2"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	s := decodeSheet(t, rec)
	if s.Lines[0].Status != "error" || s.Lines[0].Error == "" {
		t.Errorf("line 0 = %+v", s.Lines[0])
	}
	if s.Lines[1].Text != "4" {
		t.Errorf("line 1 = %+v", s.Lines[1])
	}
}

func TestSlowEvaluationTimesOut(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	h := Handler(Config{Timeout: 20 * time.Millisecond, Evaluate: func([]byte) []byte {
		<-release
		return nil
	}})
	rec := do(t, h, http.MethodPost, "/v1/sheet", `{"lines":["1"]}`)
	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status %d", rec.Code)
	}
}

func TestBusyServerRefusesWork(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	h := Handler(Config{Concurrency: 1, Timeout: time.Minute, Evaluate: func([]byte) []byte {
		started <- struct{}{}
		<-release
		return []byte(`{}`)
	}})
	first := make(chan int)
	go func() { first <- do(t, h, http.MethodPost, "/v1/sheet", `{"lines":["1"]}`).Code }()
	<-started
	rec := do(t, h, http.MethodPost, "/v1/sheet", `{"lines":["1"]}`)
	close(release)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d", rec.Code)
	}
	if code := <-first; code != http.StatusOK {
		t.Errorf("first status %d", code)
	}
}

func TestCORSIsOffByDefault(t *testing.T) {
	rec := do(t, Handler(Config{}), http.MethodGet, "/healthz", "")
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("origin header present by default")
	}
}

func TestCORSOptIn(t *testing.T) {
	h := Handler(Config{CORSOrigin: "https://calc.example"})
	rec := do(t, h, http.MethodOptions, "/v1/sheet", "")
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://calc.example" {
		t.Errorf("preflight status %d headers %v", rec.Code, rec.Header())
	}
}

func TestNewSetsEveryServerTimeout(t *testing.T) {
	s := New(Config{Addr: ":0"})
	if s.ReadHeaderTimeout <= 0 || s.ReadTimeout <= 0 || s.WriteTimeout <= 0 || s.IdleTimeout <= 0 {
		t.Errorf("timeouts = %v %v %v %v", s.ReadHeaderTimeout, s.ReadTimeout, s.WriteTimeout, s.IdleTimeout)
	}
	if s.WriteTimeout <= RequestTimeout {
		t.Errorf("write timeout %v must outlast request timeout %v", s.WriteTimeout, RequestTimeout)
	}
}
