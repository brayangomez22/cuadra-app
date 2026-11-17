package flags

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/isolated"
)

// Provider is the OpenFeature provider of Cuadra's flags: the targeting key
// is the tenant's ID and every flag is boolean. Code that asks for a flag
// uses an OpenFeature client (NewClient) with TenantContext, so the
// provider can change (a SaaS, flagd) without touching it.
type Provider struct {
	eval *Evaluator
}

var _ openfeature.FeatureProvider = (*Provider)(nil)

// NewProvider builds the provider on an Evaluator.
func NewProvider(eval *Evaluator) *Provider { return &Provider{eval: eval} }

// NewClient registers p on an OpenFeature API of its own (not the global
// singleton, so tests and several servers in one process do not share it)
// and returns its client.
func NewClient(ctx context.Context, p *Provider) (*openfeature.Client, error) {
	api := isolated.NewAPI()
	if err := api.SetProviderAndWait(ctx, p); err != nil {
		return nil, err
	}
	return api.NewClient(), nil
}

// TenantContext is the evaluation context of a tenant.
func TenantContext(tenantID uuid.UUID) openfeature.EvaluationContext {
	return openfeature.NewEvaluationContext(tenantID.String(), nil)
}

// Metadata names the provider.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "cuadra-postgres"}
}

// Hooks returns no hooks.
func (p *Provider) Hooks() []openfeature.Hook { return nil }

// BooleanEvaluation resolves a flag for the tenant of the targeting key.
// Without a valid tenant, for an undeclared flag or when the store fails, it
// returns defaultValue with the error.
func (p *Provider) BooleanEvaluation(
	ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext,
) openfeature.BoolResolutionDetail {
	fail := func(e openfeature.ResolutionError) openfeature.BoolResolutionDetail {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{ResolutionError: e, Reason: openfeature.ErrorReason},
		}
	}
	key, _ := flatCtx[openfeature.TargetingKey].(string)
	if key == "" {
		return fail(openfeature.NewTargetingKeyMissingResolutionError("the targeting key is the tenant's ID"))
	}
	tenantID, err := uuid.Parse(key)
	if err != nil {
		return fail(openfeature.NewInvalidContextResolutionError("the targeting key is not a tenant ID"))
	}
	if !p.eval.Registry.Known(flag) {
		return fail(openfeature.NewFlagNotFoundResolutionError(flag))
	}
	on, err := p.eval.enabled(ctx, tenantID, flag)
	if err != nil {
		return fail(openfeature.NewGeneralResolutionError(err.Error()))
	}
	return openfeature.BoolResolutionDetail{
		Value:                    on,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{Reason: openfeature.TargetingMatchReason},
	}
}

// StringEvaluation returns defaultValue with a type mismatch: every flag is boolean.
func (p *Provider) StringEvaluation(
	_ context.Context, flag string, defaultValue string, _ openfeature.FlattenedContext,
) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: notBoolean(flag)}
}

// FloatEvaluation returns defaultValue with a type mismatch: every flag is boolean.
func (p *Provider) FloatEvaluation(
	_ context.Context, flag string, defaultValue float64, _ openfeature.FlattenedContext,
) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: notBoolean(flag)}
}

// IntEvaluation returns defaultValue with a type mismatch: every flag is boolean.
func (p *Provider) IntEvaluation(
	_ context.Context, flag string, defaultValue int64, _ openfeature.FlattenedContext,
) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: notBoolean(flag)}
}

// ObjectEvaluation returns defaultValue with a type mismatch: every flag is boolean.
func (p *Provider) ObjectEvaluation(
	_ context.Context, flag string, defaultValue any, _ openfeature.FlattenedContext,
) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: notBoolean(flag)}
}

func notBoolean(flag string) openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError(flag + " is a boolean flag"),
		Reason:          openfeature.ErrorReason,
	}
}
