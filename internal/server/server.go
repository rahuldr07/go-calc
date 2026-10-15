package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/rahuldr07/go-calc/internal/doc"
)

const (
	MaxBodyBytes      = 1 << 20
	RequestTimeout    = 10 * time.Second
	ReadHeaderTimeout = 5 * time.Second
	ReadTimeout       = 15 * time.Second
	WriteTimeout      = 30 * time.Second
	IdleTimeout       = 60 * time.Second
	ShutdownGrace     = 15 * time.Second
	EngineHeadroom    = 10
)

const (
	pathIndex  = "/{$}"
	pathSheet  = "/v1/sheet"
	pathEval   = "/v1/eval"
	pathHealth = "/healthz"
)

type apiIndex struct {
	Name      string            `json:"name"`
	About     string            `json:"about"`
	Endpoints map[string]string `json:"endpoints"`
	Try       string            `json:"try"`
	Source    string            `json:"source"`
}

var index = apiIndex{
	Name:  "go-calc",
	About: "A calculator that shows how far to trust its answers: exact, proven, an estimate with its uncertainty, or refused with the reason.",
	Endpoints: map[string]string{
		"GET " + pathEval:   "evaluate one expression: ?expr=…&digits=12&angle=rad",
		"POST " + pathSheet: `evaluate a sheet: {"lines": [...], "digits": 12, "angle": "rad"}`,
		"GET " + pathHealth: "liveness",
	},
	Try:    pathEval + "?expr=0.1%2B0.2",
	Source: "https://github.com/rahuldr07/go-calc",
}

type Config struct {
	Addr        string
	CORSOrigin  string
	Timeout     time.Duration
	Concurrency int
	Evaluate    func([]byte, time.Time) []byte
}

type handler struct {
	cfg  Config
	busy chan struct{}
}

type failure struct {
	Error string `json:"error"`
}

type sheetRequest struct {
	Lines  []string `json:"lines"`
	Digits int      `json:"digits"`
	Angle  string   `json:"angle"`
}

func New(cfg Config) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           Handler(cfg),
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}
}

func Handler(cfg Config) http.Handler {
	if cfg.Timeout <= 0 {
		cfg.Timeout = RequestTimeout
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = runtime.NumCPU()
	}
	if cfg.Evaluate == nil {
		cfg.Evaluate = doc.EvalRequestBefore
	}
	h := &handler{cfg: cfg, busy: make(chan struct{}, cfg.Concurrency)}
	mux := http.NewServeMux()
	mux.HandleFunc(pathIndex, h.guarded(http.MethodGet, h.index))
	mux.HandleFunc(pathSheet, h.guarded(http.MethodPost, h.sheet))
	mux.HandleFunc(pathEval, h.guarded(http.MethodGet, h.eval))
	mux.HandleFunc(pathHealth, h.guarded(http.MethodGet, h.health))
	return mux
}

func (h *handler) guarded(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.allowOrigin(w)
		if r.Method == http.MethodOptions && h.cfg.CORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Methods", method)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != method {
			w.Header().Set("Allow", method)
			reject(w, http.StatusMethodNotAllowed, "this endpoint answers "+method+" only")
			return
		}
		next(w, r)
	}
}

func (h *handler) allowOrigin(w http.ResponseWriter) {
	if h.cfg.CORSOrigin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.cfg.CORSOrigin)
	w.Header().Add("Vary", "Origin")
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(index)
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "ok\n")
}

func (h *handler) sheet(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			reject(w, http.StatusRequestEntityTooLarge, "the request body is over "+strconv.Itoa(MaxBodyBytes)+" bytes")
			return
		}
		reject(w, http.StatusBadRequest, "the request body could not be read")
		return
	}
	var req sheetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		reject(w, http.StatusBadRequest, "the request body is not a sheet request: "+err.Error())
		return
	}
	h.answer(w, r.Context(), body)
}

func (h *handler) eval(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	expr := query.Get("expr")
	if expr == "" {
		reject(w, http.StatusBadRequest, "the expr parameter is required")
		return
	}
	req := sheetRequest{Lines: []string{expr}, Angle: query.Get("angle")}
	if raw := query.Get("digits"); raw != "" {
		digits, err := strconv.Atoi(raw)
		if err != nil {
			reject(w, http.StatusBadRequest, "digits must be a whole number, not "+strconv.Quote(raw))
			return
		}
		req.Digits = digits
	}
	body, err := json.Marshal(req)
	if err != nil {
		reject(w, http.StatusBadRequest, err.Error())
		return
	}
	h.answer(w, r.Context(), body)
}

func (h *handler) answer(w http.ResponseWriter, parent context.Context, body []byte) {
	select {
	case h.busy <- struct{}{}:
	default:
		w.Header().Set("Retry-After", "1")
		reject(w, http.StatusServiceUnavailable, "every evaluation slot is busy")
		return
	}
	ctx, cancel := context.WithTimeout(parent, h.cfg.Timeout)
	defer cancel()
	engineDeadline := time.Now().Add(h.cfg.Timeout - h.cfg.Timeout/EngineHeadroom)
	done := make(chan []byte, 1)
	go func() {
		defer func() { <-h.busy }()
		done <- h.cfg.Evaluate(body, engineDeadline)
	}()
	select {
	case out := <-done:
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			reject(w, http.StatusGatewayTimeout, "the evaluation did not finish in "+h.cfg.Timeout.String())
		}
	}
}

func reject(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(failure{Error: message})
}
