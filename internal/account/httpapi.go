package account

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authoidc "github.com/woodleighschool/goodies/auth/oidc"
)

// Session describes the login capability and current authenticated user.
type Session struct {
	SSOEnabled bool               `json:"sso_enabled"`
	User       *authoidc.Identity `json:"user,omitempty"`
}

type sessionOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         Session
}

// RegisterAPI mounts session and OIDC endpoints. A nil service registers schemas only.
func RegisterAPI(api huma.API, mux *http.ServeMux, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-session", Method: http.MethodGet, Path: "/api/session"},
		func(ctx context.Context, _ *struct{}) (*sessionOutput, error) {
			return &sessionOutput{CacheControl: "no-store", Body: Session{SSOEnabled: s.Enabled(), User: s.User(ctx)}}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "delete-session", Method: http.MethodDelete, Path: "/api/session", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*struct{}, error) {
			if err := s.sessions.Destroy(ctx); err != nil {
				s.logger.ErrorContext(ctx, "logout failed", "err", err)
				return nil, huma.Error500InternalServerError("Could not sign out.")
			}
			return nil, nil
		})
	mux.HandleFunc("GET /api/auth/sso/start", s.start)
	mux.HandleFunc("GET /api/auth/sso/callback", s.callback)
}

// RequireUser gates application APIs when SSO is configured.
func (s *Service) RequireUser(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if s.Enabled() && s.User(ctx.Context()) == nil {
			ctx.SetHeader("Cache-Control", "no-store")
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
			return
		}
		next(ctx)
	}
}
