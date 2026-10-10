// Package identity wires the identity module: tenants, users, sign-up, login
// and sessions. cmd/api builds it with New and mounts its routes.
package identity

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/argon2id"
	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/ratelimit"
)

// Login rate limits. Per IP they stop a single client from spraying
// passwords across accounts; per email they stop a distributed guess of one
// account. They count every attempt, successful or not.
const (
	loginsPerIP          = 20
	loginsPerIPWindow    = time.Minute
	loginsPerEmail       = 10
	loginsPerEmailWindow = 15 * time.Minute
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Issuer issues the access tokens (*auth.JWT).
	Issuer         app.AccessTokenIssuer
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired identity module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL, Argon2id and in-memory login limits.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Issuer == nil || cfg.Logger == nil {
		return nil, errors.New("identity: missing module dependency")
	}
	users := postgres.NewUserRepository(cfg.DB)
	svc, err := app.NewService(app.Deps{
		Tenants:        postgres.NewTenantRepository(cfg.DB),
		Users:          users,
		Tokens:         postgres.NewRefreshTokenRepository(cfg.DB),
		Directory:      postgres.NewLoginDirectory(cfg.DB),
		Hasher:         argon2id.New(argon2id.DefaultParams),
		Issuer:         cfg.Issuer,
		IPLimiter:      ratelimit.NewFixedWindow(loginsPerIP, loginsPerIPWindow),
		EmailLimiter:   ratelimit.NewFixedWindow(loginsPerEmail, loginsPerEmailWindow),
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "auth") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}

// Authorize reports whether role grants permission, by the identity
// permission matrix. It is the auth.Authorizer of the API.
func Authorize(role, permission string) bool {
	r, err := domain.ParseRole(role)
	return err == nil && r.Can(domain.Permission(permission))
}
