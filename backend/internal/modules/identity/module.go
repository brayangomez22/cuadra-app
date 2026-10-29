// Package identity wires the identity module: tenants, users, sign-up, login
// and sessions. internal/server builds it with New and mounts its routes.
package identity

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
	Issuer app.AccessTokenIssuer
	// Setup prepares a new tenant's data in the other modules at sign-up.
	Setup app.TenantSetup
	// Permissions is the role → permission matrix; Modules, the active
	// modules of each tenant.
	Permissions *Permissions
	Modules     app.ActiveModules
	// Now is the clock of the use cases; nil is time.Now (the demo generator
	// simulates one).
	Now            func() time.Time
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
	if cfg.DB == nil || cfg.Issuer == nil || cfg.Setup == nil || cfg.Permissions == nil || cfg.Modules == nil || cfg.Logger == nil {
		return nil, errors.New("identity: missing module dependency")
	}
	users := postgres.NewUserRepository(cfg.DB)
	svc, err := app.NewService(app.Deps{
		Tenants:        postgres.NewTenantRepository(cfg.DB),
		Users:          users,
		Tokens:         postgres.NewRefreshTokenRepository(cfg.DB),
		Directory:      postgres.NewLoginDirectory(cfg.DB),
		Demo:           postgres.NewDemoTenantRepository(cfg.DB),
		Hasher:         argon2id.New(argon2id.DefaultParams),
		Issuer:         cfg.Issuer,
		Setup:          cfg.Setup,
		Permissions:    cfg.Permissions.matrix,
		Modules:        cfg.Modules,
		IPLimiter:      ratelimit.NewFixedWindow(loginsPerIP, loginsPerIPWindow),
		EmailLimiter:   ratelimit.NewFixedWindow(loginsPerEmail, loginsPerEmailWindow),
		Now:            cfg.Now,
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

// DefaultTaxRate returns the IVA tenantID proposes to a new product (see
// app.Service.DefaultTaxRate). It is for the imports module.
func (m *Module) DefaultTaxRate(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return m.service.DefaultTaxRate(ctx, tenantID)
}

// Permissions is the permission matrix of the core and the registered
// modules. It is built once, at startup, before the modules that authorize
// with it.
type Permissions struct{ matrix *domain.PermissionMatrix }

// NewPermissions builds the matrix from the module registry. It fails if a
// module repeats or misspells a permission, or names an unknown role.
func NewPermissions(registry app.ModuleRegistry) (*Permissions, error) {
	matrix, err := app.NewPermissionMatrix(registry)
	if err != nil {
		return nil, err
	}
	return &Permissions{matrix: matrix}, nil
}

// Authorize reports whether role grants permission. An unknown role or an
// unregistered permission grants nothing. It is the auth.Authorizer of the
// API.
func (p *Permissions) Authorize(role, permission string) bool {
	r, err := domain.ParseRole(role)
	return err == nil && p.matrix.Can(r, domain.Permission(permission))
}

// Known reports whether permission is registered, by the core or a module.
func (p *Permissions) Known(permission string) bool {
	return p.matrix.Known(domain.Permission(permission))
}

// AuthorizeAction returns the user of tenantID with the credentials if its
// role grants permission (see app.Service.AuthorizeAction). It is for the
// sales module, which asks it who authorizes a sale over a customer's
// credit, through the port internal/server adapts.
func (m *Module) AuthorizeAction(ctx context.Context, tenantID uuid.UUID, email, password string, permission domain.Permission) (uuid.UUID, error) {
	return m.service.AuthorizeAction(ctx, app.AuthorizationInput{
		TenantID: tenantID, Email: email, Password: password, Permission: permission,
	})
}
