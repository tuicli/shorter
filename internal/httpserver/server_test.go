package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tuicli/shorter/internal/app"
)

func TestRedirectHandler(t *testing.T) {
	service := fakeRedirectService{
		results: map[string]app.RedirectResult{
			"ABC123": {
				Status:      app.RedirectFound,
				HTTPStatus:  http.StatusFound,
				OriginalURL: "https://example.com/a",
				ShortLinkID: 1,
			},
			"OFF123": {
				Status:      app.RedirectDisabled,
				HTTPStatus:  http.StatusNotFound,
				ShortLinkID: 2,
			},
		},
	}
	handler := New(service, nil)

	t.Run("active redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ABC123", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "https://example.com/a" {
			t.Fatalf("location = %q", got)
		}
	})

	t.Run("disabled is not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/OFF123", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("invalid nested path is not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ABC123/extra", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}

type fakeRedirectService struct {
	results map[string]app.RedirectResult
}

func (s fakeRedirectService) ResolveRedirect(_ context.Context, code string) (app.RedirectResult, error) {
	result, ok := s.results[code]
	if !ok {
		return app.RedirectResult{Status: app.RedirectNotFound, HTTPStatus: http.StatusNotFound}, nil
	}
	return result, nil
}
