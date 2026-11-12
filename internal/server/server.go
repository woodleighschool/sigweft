// Package server exposes signature generation and serves the web application.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/woodleighschool/sigweft/internal/account"
	"github.com/woodleighschool/sigweft/internal/signature"
)

type formOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         signature.Form
}

type renderInput struct {
	Body struct {
		Values map[string]any `json:"values"`
	}
}

type renderOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         signature.Result
}

// New registers the same API for the running server and generated OpenAPI document.
func New(renderer *signature.Renderer, web fs.FS, logger *slog.Logger, version string, auth *account.Service) (http.Handler, huma.API) {
	mux := http.NewServeMux()
	cfg := huma.DefaultConfig("Sigweft", version)
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	api := humago.New(mux, cfg)
	account.RegisterAPI(api, mux, auth)
	protected := huma.NewGroup(api)
	protected.UseMiddleware(auth.RequireUser(api))
	huma.Register(protected, huma.Operation{OperationID: "get-form", Method: http.MethodGet, Path: "/api/form", Errors: []int{http.StatusUnauthorized}},
		func(_ context.Context, _ *struct{}) (*formOutput, error) {
			return &formOutput{CacheControl: "no-store", Body: renderer.Form()}, nil
		})
	huma.Register(protected, huma.Operation{OperationID: "render-signature", Method: http.MethodPost, Path: "/api/render", MaxBodyBytes: 64 << 10, Errors: []int{http.StatusUnauthorized}},
		func(ctx context.Context, input *renderInput) (*renderOutput, error) {
			result, err := renderer.Render(input.Body.Values)
			if err != nil {
				logger.ErrorContext(ctx, "render signature", "error", err)
				return nil, huma.Error500InternalServerError("Could not render the signature. Check the configured template.")
			}
			return &renderOutput{CacheControl: "no-store", Body: result}, nil
		})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	if web != nil {
		files := http.FileServerFS(web)
		mux.Handle("GET /", files)
		mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = "/"
			files.ServeHTTP(w, r)
		})
	}
	if auth != nil {
		return auth.LoadAndSave(mux), api
	}
	return mux, api
}
