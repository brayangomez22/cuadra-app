// Package httpadapter exposes the identity use cases over HTTP: it
// implements the strict server generated from the "auth" operations of
// api/openapi.yaml (identityapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// UseCases are the identity use cases the handlers call (*app.Service).
type UseCases interface {
	SignUp(ctx context.Context, in app.SignUpInput) (app.Session, error)
	Login(ctx context.Context, in app.LoginInput) (app.Session, error)
	Refresh(ctx context.Context, raw string) (app.Session, error)
	Logout(ctx context.Context, raw string) error
	Me(ctx context.Context, tenantID, userID uuid.UUID) (app.Profile, error)
}

// Handler implements identityapi.StrictServerInterface.
type Handler struct {
	uc  UseCases
	now func() time.Time
}

var _ identityapi.StrictServerInterface = (*Handler)(nil)

// Option configures the handler.
type Option func(*Handler)

// WithClock replaces time.Now, used to compute expires_in and cookie ages (tests).
func WithClock(now func() time.Time) Option { return func(h *Handler) { h.now = now } }

// Register mounts the identity operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger, opts ...Option) {
	h := &Handler{uc: uc, now: time.Now}
	for _, opt := range opts {
		opt(h)
	}
	errs := httpx.NewErrorMapper(log, errorRules...)
	identityapi.HandlerWithOptions(
		identityapi.NewStrictHandlerWithOptions(h, nil, identityapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		identityapi.StdHTTPServerOptions{
			BaseRouter:       mux,
			ErrorHandlerFunc: errs.RequestError,
			Middlewares:      []identityapi.MiddlewareFunc{withClientIP},
		},
	)
}

// SignUp registers a business and its owner, and starts a session.
func (h *Handler) SignUp(ctx context.Context, req identityapi.SignUpRequestObject) (identityapi.SignUpResponseObject, error) {
	session, err := h.uc.SignUp(ctx, app.SignUpInput{
		TenantName: req.Body.TenantName,
		NIT:        req.Body.Nit,
		OwnerName:  req.Body.Name,
		Email:      req.Body.Email,
		Password:   req.Body.Password,
	})
	if err != nil {
		return nil, err
	}
	body, cookie := h.authSession(session)
	return identityapi.SignUp201JSONResponse{Body: body, Headers: identityapi.SignUp201ResponseHeaders{SetCookie: &cookie}}, nil
}

// Login checks the credentials and starts a session.
func (h *Handler) Login(ctx context.Context, req identityapi.LoginRequestObject) (identityapi.LoginResponseObject, error) {
	in := app.LoginInput{Email: req.Body.Email, Password: req.Body.Password, ClientIP: clientIP(ctx)}
	if req.Body.TenantId != nil {
		in.TenantID = *req.Body.TenantId
	}
	session, err := h.uc.Login(ctx, in)

	var selection *app.TenantSelectionError
	var limited *app.RateLimitError
	switch {
	case errors.As(err, &selection):
		tenants := make([]identityapi.TenantOption, 0, len(selection.Tenants))
		for _, t := range selection.Tenants {
			tenants = append(tenants, identityapi.TenantOption{Id: t.ID, Name: t.Name})
		}
		return identityapi.Login409JSONResponse{
			Error:   identityapi.ErrorDetail{Code: "tenant_selection_required", Message: tenantSelectionMessage},
			Tenants: tenants,
		}, nil
	case errors.As(err, &limited):
		retryAfter := int(math.Ceil(limited.RetryAfter.Seconds()))
		return identityapi.Login429JSONResponse{
			Body:    errorBody("too_many_requests", tooManyRequestsMessage),
			Headers: identityapi.Login429ResponseHeaders{RetryAfter: &retryAfter},
		}, nil
	case err != nil:
		return nil, err
	}

	body, cookie := h.authSession(session)
	return identityapi.Login200JSONResponse{Body: body, Headers: identityapi.Login200ResponseHeaders{SetCookie: &cookie}}, nil
}

// RefreshSession rotates the refresh token of the cookie. Any session error
// deletes the cookie: it can never be used again.
func (h *Handler) RefreshSession(ctx context.Context, req identityapi.RefreshSessionRequestObject) (identityapi.RefreshSessionResponseObject, error) {
	if req.Params.CuadraRefresh == nil || *req.Params.CuadraRefresh == "" {
		return invalidSession(), nil
	}
	session, err := h.uc.Refresh(ctx, *req.Params.CuadraRefresh)
	switch {
	case errors.Is(err, domain.ErrInvalidRefreshToken),
		errors.Is(err, domain.ErrRefreshTokenExpired),
		errors.Is(err, domain.ErrRefreshTokenReused):
		return invalidSession(), nil
	case err != nil:
		return nil, err
	}
	body, cookie := h.authSession(session)
	return identityapi.RefreshSession200JSONResponse{Body: body, Headers: identityapi.RefreshSession200ResponseHeaders{SetCookie: &cookie}}, nil
}

// Logout revokes the cookie's session and deletes the cookie.
func (h *Handler) Logout(ctx context.Context, req identityapi.LogoutRequestObject) (identityapi.LogoutResponseObject, error) {
	var raw string
	if req.Params.CuadraRefresh != nil {
		raw = *req.Params.CuadraRefresh
	}
	if err := h.uc.Logout(ctx, raw); err != nil {
		return nil, err
	}
	cleared := clearedRefreshCookie()
	return identityapi.Logout204Response{Headers: identityapi.Logout204ResponseHeaders{SetCookie: &cleared}}, nil
}

// GetMe returns the authenticated user. Its tenant and id come only from the
// access token (the principal set by auth.RequireAuth).
func (h *Handler) GetMe(ctx context.Context, _ identityapi.GetMeRequestObject) (identityapi.GetMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return identityapi.GetMe401JSONResponse{UnauthorizedJSONResponse: identityapi.UnauthorizedJSONResponse(errorBody("unauthorized", unauthorizedMessage))}, nil
	}
	profile, err := h.uc.Me(ctx, p.TenantID, p.UserID)
	if err != nil {
		return nil, err
	}
	return identityapi.GetMe200JSONResponse(me(profile)), nil
}

// authSession builds the response body and the refresh cookie of a session.
func (h *Handler) authSession(s app.Session) (identityapi.AuthSession, string) {
	now := h.now()
	return identityapi.AuthSession{
		AccessToken: s.AccessToken,
		TokenType:   identityapi.Bearer,
		ExpiresIn:   int(s.AccessExpiresAt.Sub(now).Round(time.Second).Seconds()),
		User:        me(s.Profile),
	}, refreshCookie(s.RefreshToken, s.RefreshExpiresAt, now)
}

func me(p app.Profile) identityapi.Me {
	permissions := p.User.Role().Permissions()
	names := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		names = append(names, string(perm))
	}
	return identityapi.Me{
		Id:          p.User.ID(),
		TenantId:    p.Tenant.ID(),
		TenantName:  p.Tenant.Name(),
		Name:        p.User.Name(),
		Email:       p.User.Email().String(),
		Role:        identityapi.MeRole(p.User.Role()),
		Permissions: names,
	}
}

func invalidSession() identityapi.RefreshSession401JSONResponse {
	cleared := clearedRefreshCookie()
	return identityapi.RefreshSession401JSONResponse{
		Body:    errorBody("invalid_session", invalidSessionMessage),
		Headers: identityapi.RefreshSession401ResponseHeaders{SetCookie: &cleared},
	}
}

func errorBody(code, message string) identityapi.Error {
	return identityapi.Error{Error: identityapi.ErrorDetail{Code: code, Message: message}}
}

type clientIPKey struct{}

// withClientIP stores the client IP (RemoteAddr without the port) for the
// login rate limit. Forwarded headers are not trusted: behind a proxy, the
// trusted-proxy configuration belongs to the hardening task (T28).
func withClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
	})
}

func clientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}
