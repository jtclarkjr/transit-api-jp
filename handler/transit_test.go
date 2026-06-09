package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransitRejectsInvalidStartTime(t *testing.T) {
	t.Setenv("OPENAI_PROXY_APP_TOKEN", "secret")

	req := httptest.NewRequest("GET", "http://localhost:8080/transit?lang=en&ai_translate=true&start=東京駅&goal=新宿駅&start_time=2026-06-09T018:01:00", nil)
	req.RemoteAddr = "127.0.0.1:51234"
	req.Header.Set(appTokenHeader, "secret")
	recorder := httptest.NewRecorder()

	Transit()(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "start_time") {
		t.Fatalf("body = %q, want start_time validation error", recorder.Body.String())
	}
}

func TestBuildTransitRouteURLEncodesStartTime(t *testing.T) {
	got := buildTransitRouteURL("example.test", "00004212", "00005975", "2026-06-09T18:01:00")

	if !strings.Contains(got, "start_time=2026-06-09T18%3A01%3A00") {
		t.Fatalf("URL = %q, want encoded start_time query", got)
	}
}
