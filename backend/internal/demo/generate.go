package demo

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/server"
)

// tokenTTL outlives any generation in simulated time: the generator logs in
// once, on the first day, and writes six months with the same tokens.
const tokenTTL = 400 * 24 * time.Hour

// Generator writes scenarios into the database through an API of its own,
// built on the simulated clock of each generation.
type Generator struct {
	DB *db.DB
	// Jobs is the process's job client: the API needs one (imports), the
	// generator enqueues nothing.
	Jobs           *jobs.Client
	Log            *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// runner is one generation: the API on its clock, the users' sessions and
// the ids of what it created, by the scenario's indexes.
type runner struct {
	api      *apiclient.ClientWithResponses
	demo     *server.Demo
	clock    *Clock
	s        *Scenario
	tokens   map[string]string
	ids      ids
	sessions []uuid.UUID
	expected []expectedCash
}

// Generate signs up the demo tenant of s with its staff (all with
// password) and writes its catalog, customers, suppliers and every day of
// its history; it returns the tenant's id.
func (g Generator) Generate(ctx context.Context, s Scenario, password string) (uuid.UUID, error) {
	r, err := g.newRunner(&s)
	if err != nil {
		return uuid.Nil, err
	}
	r.clock.Set(at(s.Days[0].Date, openingHour*60-60))
	tenant, err := r.demo.SetUpTenant(ctx, server.DemoTenant{
		TenantName: s.Profile.Tenant.Name, NIT: withCheckDigit(s.Profile.Tenant.NIT),
		BusinessTemplate: s.Profile.Template, Password: password,
		Owner: server.DemoUser{Name: "Dueño Demo", Email: s.Profile.OwnerEmail()},
		Staff: s.Profile.Staff(),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("demo: set up tenant: %w", err)
	}
	if err := r.logIn(ctx, tenant, password); err != nil {
		return tenant, err
	}
	if err := r.writeCatalog(ctx); err != nil {
		return tenant, err
	}
	if err := r.writeReorderRules(ctx); err != nil {
		return tenant, err
	}
	if err := r.writePeople(ctx); err != nil {
		return tenant, err
	}
	for i := range s.Days {
		if err := r.writeDay(ctx, &s.Days[i]); err != nil {
			return tenant, fmt.Errorf("demo: day %s: %w", s.Days[i].Date.Format(time.DateOnly), err)
		}
	}
	return tenant, nil
}

// newRunner builds the generator's API on a new clock.
func (g Generator) newRunner(s *Scenario) (*runner, error) {
	clock := &Clock{}
	h, d, err := g.newAPI(clock)
	if err != nil {
		return nil, err
	}
	api, err := apiclient.NewClientWithResponses("http://demo.internal", apiclient.WithHTTPClient(handlerDoer{h}))
	if err != nil {
		return nil, err
	}
	return &runner{api: api, demo: d, clock: clock, s: s, tokens: map[string]string{}}, nil
}

// newAPI builds an API on clock with a signing key of its own: its tokens
// are good only for it.
func (g Generator) newAPI(clock *Clock) (http.Handler, *server.Demo, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, nil, err
	}
	tokens, err := auth.NewJWT(secret, tokenTTL, auth.WithClock(clock.Now))
	if err != nil {
		return nil, nil, err
	}
	return server.NewDemo(server.Config{
		DB: g.DB, Tokens: tokens, Jobs: g.Jobs, Workers: river.NewWorkers(), Log: g.Log,
		TracerProvider: g.TracerProvider, MeterProvider: g.MeterProvider, Now: clock.Now,
	})
}

// logIn starts a session for the owner and each member of the staff.
func (r *runner) logIn(ctx context.Context, tenant uuid.UUID, password string) error {
	emails := []string{r.s.Profile.OwnerEmail()}
	for _, u := range r.s.Profile.Staff() {
		emails = append(emails, u.Email)
	}
	for _, email := range emails {
		res, err := r.api.LoginWithResponse(ctx, apiclient.LoginRequest{Email: email, Password: password, TenantId: &tenant})
		if err := expect("log in", res, err); err != nil {
			return err
		}
		r.tokens[email] = res.JSON200.AccessToken
	}
	return nil
}

// as authenticates a request as the user with email.
func (r *runner) as(email string) apiclient.RequestEditorFn {
	token := r.tokens[email]
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// response is what every generated response has.
type response interface{ StatusCode() int }

// expect turns a failed call or a response that is not 2xx into an error.
func expect[R response](op string, res R, err error) error {
	if err != nil {
		return fmt.Errorf("demo: %s: %w", op, err)
	}
	if code := res.StatusCode(); code < 200 || code > 299 {
		return fmt.Errorf("demo: %s: status %d: %s", op, code, body(res))
	}
	return nil
}

// body returns the Body field every generated response keeps.
func body(res any) string {
	v := reflect.ValueOf(res)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		if f := v.Elem().FieldByName("Body"); f.IsValid() {
			return string(f.Bytes())
		}
	}
	return ""
}
