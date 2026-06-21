package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tuicli/shorter/internal/app"
)

type RedirectService interface {
	ResolveRedirect(ctx context.Context, code string) (app.RedirectResult, error)
}

func New(service RedirectService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", handleRedirect(service, logger))

	return mux
}

func handleRedirect(service RedirectService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		code := strings.TrimPrefix(r.URL.Path, "/")
		requestID := r.Header.Get("X-Request-ID")
		result := app.RedirectInvalid
		status := http.StatusNotFound
		var shortLinkID int64
		var err error

		defer func() {
			attrs := []any{
				"code", code,
				"result", string(result),
				"http_status", status,
				"elapsed_ms", time.Since(started).Milliseconds(),
			}
			if requestID != "" {
				attrs = append(attrs, "request_id", requestID)
			}
			if shortLinkID != 0 {
				attrs = append(attrs, "short_link_id", shortLinkID)
			}
			if err != nil {
				logger.Error("redirect request failed", append(attrs, "error", err)...)
				return
			}
			logger.Info("redirect request finished", attrs...)
		}()

		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != http.MethodGet {
			status = http.StatusMethodNotAllowed
			w.WriteHeader(status)
			return
		}
		if code == "" || strings.Contains(code, "/") {
			http.NotFound(w, r)
			return
		}

		redirect, resolveErr := service.ResolveRedirect(r.Context(), code)
		err = resolveErr
		if resolveErr != nil {
			result = app.RedirectNotFound
			status = http.StatusInternalServerError
			http.Error(w, "internal server error", status)
			return
		}

		result = redirect.Status
		status = redirect.HTTPStatus
		shortLinkID = redirect.ShortLinkID
		if redirect.Status != app.RedirectFound {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, redirect.OriginalURL, http.StatusFound)
	}
}
