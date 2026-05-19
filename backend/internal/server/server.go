package server

import (
	"log/slog"
	"net/http"
	"time"
	"tracker/internal/handler"

	"github.com/go-chi/chi/v5"
)

func New(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(recoveryMiddleware)
	r.Use(requestLogger)

	r.MethodFunc(http.MethodOptions, "/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/config/{clientId}", h.GetConfig)
	r.Post("/events", h.PostEvents)
	r.Get("/sessions", h.GetSessions)
	r.Get("/sessions/{sessionId}", h.GetSessionByID)
	r.Get("/stats/{clientId}", h.GetStats)
	r.Get("/widget.js", h.ServeWidgetJS)
	r.Get("/dashboard", h.ServeDashboard)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"error", rec,
					"method", r.Method,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"status":"error","message":"internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	return sr.ResponseWriter.Write(b)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		args := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		switch {
		case rec.status >= 500:
			slog.Error("request", args...)
		case rec.status >= 400:
			slog.Warn("request", args...)
		default:
			slog.Info("request", args...)
		}
	})
}
