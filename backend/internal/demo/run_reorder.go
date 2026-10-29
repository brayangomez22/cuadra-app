package demo

import (
	"context"
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
)

// writeReorderRules sets the minimum and maximum stock of the scenario's
// rules, as the owner.
func (r *runner) writeReorderRules(ctx context.Context) error {
	owner := r.as(r.s.Profile.OwnerEmail())
	for _, rule := range r.s.Reorder {
		res, err := r.api.SetReorderRuleWithResponse(ctx, r.ids.locations[rule.Location], r.ids.variants[rule.Product][rule.Variant],
			apiclient.ReorderRuleInput{MinQuantity: rule.Min.String(), MaxQuantity: rule.Max.String()}, owner)
		if err := expect("set reorder rule", res, err); err != nil {
			return fmt.Errorf("product %q: %w", r.s.Products[rule.Product].Name, err)
		}
	}
	return nil
}
