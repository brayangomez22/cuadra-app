package domain

import (
	"fmt"
	"slices"
	"unicode/utf8"
)

// Action is what an audited operation did, from the closed catalog of
// docs/auditoria.md. Its names are neutral: they fit any business type.
type Action string

// CatalogVersion is the version of the catalog in docs/auditoria.md. A new
// action, or a change to the details of one, raises it.
const CatalogVersion = 2

// MaxDetailLen is the longest value of details, in characters: they hold
// figures, codes and settings, not prose.
const MaxDetailLen = 200

// The catalog's actions.
const (
	ActionSaleVoided               Action = "sale.voided"
	ActionSaleDiscountApplied      Action = "sale.discount_applied"
	ActionVariantPriceChanged      Action = "variant.price_changed"
	ActionStockAdjusted            Action = "stock.adjusted"
	ActionCashClosedWithDifference Action = "cash.closed_with_difference"
	ActionPurchaseVoided           Action = "purchase.voided"
	ActionModuleEnabled            Action = "module.enabled"
	ActionModuleDisabled           Action = "module.disabled"
	ActionTenantSettingsChanged    Action = "tenant.settings_changed"
	ActionUserInvited              Action = "user.invited"
	ActionUserRoleChanged          Action = "user.role_changed"
	ActionUserDeactivated          Action = "user.deactivated"
	ActionUserActivated            Action = "user.activated"
	ActionUserPasswordReset        Action = "user.password_reset"
	ActionCreditTermsChanged       Action = "credit.terms_changed"
)

// actionSpec is what an action is about and the details it carries: every
// key is required and no other is accepted.
type actionSpec struct {
	entityType string
	// hasEntityID is false when the entity has no uuid (a module).
	hasEntityID bool
	keys        []string
}

var catalog = map[Action]actionSpec{
	ActionSaleVoided:               {"sale", true, []string{"total"}},
	ActionSaleDiscountApplied:      {"sale", true, []string{"discount", "gross"}},
	ActionVariantPriceChanged:      {"variant", true, []string{"new_price", "old_price"}},
	ActionStockAdjusted:            {"variant", true, []string{"quantity", "source"}},
	ActionCashClosedWithDifference: {"cash_session", true, []string{"counted", "difference", "expected"}},
	ActionPurchaseVoided:           {"purchase", true, []string{"total"}},
	ActionModuleEnabled:            {"module", false, []string{"module"}},
	ActionModuleDisabled:           {"module", false, []string{"module"}},
	ActionTenantSettingsChanged:    {"tenant", true, []string{"new", "old", "setting"}},
	ActionUserInvited:              {"invitation", true, []string{"role"}},
	ActionUserRoleChanged:          {"user", true, []string{"new_role", "old_role"}},
	ActionUserDeactivated:          {"user", true, nil},
	ActionUserActivated:            {"user", true, nil},
	ActionUserPasswordReset:        {"user", true, nil},
	ActionCreditTermsChanged:       {"customer", true, []string{"new_limit", "new_term_days", "old_limit", "old_term_days"}},
}

// ParseAction returns the catalog's action named s, or ErrUnknownAction.
func ParseAction(s string) (Action, error) {
	a := Action(s)
	if _, ok := catalog[a]; !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownAction, s)
	}
	return a, nil
}

// Actions returns the catalog's actions in alphabetical order.
func Actions() []Action {
	out := make([]Action, 0, len(catalog))
	for a := range catalog {
		out = append(out, a)
	}
	slices.Sort(out)
	return out
}

// EntityType is the kind of entity the action is about ("sale",
// "variant"...); empty for an action outside the catalog.
func (a Action) EntityType() string { return catalog[a].entityType }

// checkDetails tells whether details has exactly the keys of the action,
// each with a value of up to MaxDetailLen characters (empty is a setting
// that was cleared).
func (s actionSpec) checkDetails(details map[string]string) error {
	if len(details) != len(s.keys) {
		return fmt.Errorf("%w: want keys %v", ErrInvalidDetails, s.keys)
	}
	for _, k := range s.keys {
		v, ok := details[k]
		if !ok || utf8.RuneCountInString(v) > MaxDetailLen {
			return fmt.Errorf("%w: key %q", ErrInvalidDetails, k)
		}
	}
	return nil
}
