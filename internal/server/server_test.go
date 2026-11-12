package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/woodleighschool/sigweft/internal/signature"
)

func TestFormAndRenderContract(t *testing.T) {
	renderer, err := signature.Load(t.Context(), "../../examples/studio/sigweft.yaml")
	if err != nil {
		t.Fatal(err)
	}
	handler, _ := New(renderer, nil, slog.New(slog.DiscardHandler), "test", nil)
	form := httptest.NewRecorder()
	handler.ServeHTTP(form, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/form", nil))
	if form.Code != 200 || form.Header().Get("Cache-Control") != "no-store" || strings.Contains(form.Body.String(), "12 Example Street") {
		t.Fatalf("unexpected form response: %d %s", form.Code, form.Body.String())
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/render", strings.NewReader(`{"values":{"display_name":"Jo Example","contact":"jo@example.org","disciplines":["Designer"],"base":"riverside"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var result signature.Result
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(result.Errors) != 0 || !strings.Contains(result.Text, "+1 202 555 0102") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected rendering: %d %+v", response.Code, result.Errors)
	}
	for _, body := range []string{`{"values":42}`, `{"values":{"name":"` + strings.Repeat("a", 70<<10) + `"}}`} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/render", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code < 400 || response.Code >= 500 {
			t.Errorf("invalid request status %d", response.Code)
		}
	}
}
