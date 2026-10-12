package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/proof"
	"github.com/canadian-ai/girl/internal/shared"
	"github.com/urfave/cli/v2"
)

// ServeCommand exposes GIRL over a local HTTP API. A long-lived process lets
// editors and agents skip process start-up per query, and gives load tools
// such as k6 something to drive when comparing hardware profiles.
func ServeCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Serve GIRL analysis over a local HTTP API (for agents, editors, and k6 load tests)",
		Description: "Endpoints: GET /healthz, GET /v1/capabilities, POST /v1/next, POST /v1/benchmark.\n" +
			"Request paths are resolved inside --root; anything outside it is rejected.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "addr", Value: "127.0.0.1:7777", Usage: "Listen address (loopback by default)"},
			&cli.StringFlag{Name: "root", Value: ".", Usage: "Directory requests may analyze"},
			&cli.IntFlag{Name: "max-concurrent", Value: 2, Usage: "Analyses run at once; further requests queue"},
		},
		Action: func(c *cli.Context) error {
			srv, err := newAnalysisServer(stringFlag(c, "root"), intFlag(c, "max-concurrent"), c.App)
			if err != nil {
				return err
			}
			addr := stringFlag(c, "addr")
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			httpSrv := &http.Server{Handler: srv.routes(), ReadHeaderTimeout: 10 * time.Second}
			fmt.Fprintf(os.Stderr, "girl serve: http://%s (root %s, %d concurrent, %d workers each)\n",
				ln.Addr(), shared.DisplayTarget(srv.root), cap(srv.slots), shared.Workers())

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			go func() {
				<-ctx.Done()
				shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = httpSrv.Shutdown(shutdown)
			}()
			if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
}

type analysisServer struct {
	root  string
	slots chan struct{}
	caps  Capabilities
}

func newAnalysisServer(root string, maxConcurrent int, app *cli.App) (*analysisServer, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if abs, err = filepath.EvalSymlinks(abs); err != nil {
		return nil, fmt.Errorf("serve root: %w", err)
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	s := &analysisServer{root: abs, slots: make(chan struct{}, maxConcurrent)}
	if app != nil {
		s.caps = BuildCapabilities(app)
	}
	return s, nil
}

func (s *analysisServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "workers": shared.Workers(), "maxConcurrent": cap(s.slots)})
	})
	mux.HandleFunc("GET /v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.caps)
	})
	mux.HandleFunc("POST /v1/next", s.handleNext)
	mux.HandleFunc("POST /v1/benchmark", s.handleBenchmark)
	return mux
}

// NextRequest mirrors the `girl next` flags.
type NextRequest struct {
	Path         string `json:"path"`
	Limit        int    `json:"limit"`
	Changed      bool   `json:"changed"`
	Base         string `json:"base"`
	MinSeverity  string `json:"minSeverity"`
	ExcerptLines *int   `json:"excerptLines"`
}

func (s *analysisServer) handleNext(w http.ResponseWriter, r *http.Request) {
	var req NextRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	target, ok := s.resolve(w, req.Path)
	if !ok {
		return
	}
	excerpt := 12
	if req.ExcerptLines != nil {
		excerpt = *req.ExcerptLines
	}
	s.run(w, r, func() (any, int, error) {
		res, err := buildNext(target, nextOptions{
			Limit:        req.Limit,
			Changed:      req.Changed || req.Base != "",
			Base:         req.Base,
			MinSeverity:  ir.Severity(strings.ToLower(req.MinSeverity)),
			ExcerptLines: excerpt,
		})
		if err != nil {
			return nil, 0, err
		}
		return res, res.Summary.Files, nil
	})
}

// BenchmarkRequest asks for repository-wide summary counts.
type BenchmarkRequest struct {
	Path     string `json:"path"`
	Lang     string `json:"lang"`
	TopFiles int    `json:"topFiles"`
}

func (s *analysisServer) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	target, ok := s.resolve(w, req.Path)
	if !ok {
		return
	}
	if req.Lang == "" {
		req.Lang = "auto"
	}
	if req.TopFiles <= 0 {
		req.TopFiles = 10
	}
	s.run(w, r, func() (any, int, error) {
		result, err := analyzePath(target, resolveLang(target, req.Lang))
		if err != nil {
			return nil, 0, err
		}
		return proof.NewBenchmarkReport(target, result, req.TopFiles), len(result.Files), nil
	})
}

// run waits for an analysis slot (or the client giving up), runs fn, and
// reports timing in headers so load tests can separate queueing from work.
func (s *analysisServer) run(w http.ResponseWriter, r *http.Request, fn func() (any, int, error)) {
	queued := time.Now()
	select {
	case s.slots <- struct{}{}:
	case <-r.Context().Done():
		writeError(w, http.StatusServiceUnavailable, "request cancelled while queued")
		return
	}
	defer func() { <-s.slots }()

	started := time.Now()
	body, files, err := fn()
	elapsed := time.Since(started)
	w.Header().Set("Server-Timing", fmt.Sprintf("queue;dur=%.1f, analysis;dur=%.1f",
		ms(started.Sub(queued)), ms(elapsed)))
	w.Header().Set("X-Girl-Files", strconv.Itoa(files))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// resolve maps a request path to a directory or file inside the serve root.
func (s *analysisServer) resolve(w http.ResponseWriter, path string) (string, bool) {
	if path == "" {
		path = "."
	}
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.root, candidate)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(candidate))
	if err != nil {
		writeError(w, http.StatusNotFound, "path not found")
		return "", false
	}
	if resolved != s.root && !strings.HasPrefix(resolved, s.root+string(filepath.Separator)) {
		writeError(w, http.StatusForbidden, "path is outside the serve root")
		return "", false
	}
	return resolved, true
}

func decodeRequest(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
