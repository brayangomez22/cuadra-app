package domain

import (
	"fmt"
	"slices"
)

// StepID names a step of the catalog. Templates list them in their
// getting_started key.
type StepID string

// The steps of the catalog. The names are neutral: a template picks the
// ones that fit its kind of business.
const (
	StepBusinessProfile StepID = "business_profile" // the address, city and phone of the business
	StepInvoicing       StepID = "invoicing"        // a numbering resolution in force for electronic invoicing
	StepProducts        StepID = "products"         // the first product, created or imported
	StepAlternateUnits  StepID = "alternate_units"  // a variant sold in another unit (module unit_conversions)
	StepVariantProducts StepID = "variant_products" // a product with variants (module variants)
	StepInitialStock    StepID = "initial_stock"    // the first stock movement
	StepReorderRules    StepID = "reorder_rules"    // a minimum stock to reorder
	StepCustomerCredit  StepID = "customer_credit"  // a customer with a credit limit
	StepInviteTeam      StepID = "invite_team"      // another user, or an invitation in force
	StepFirstSale       StepID = "first_sale"       // the first sale, which needs the cash drawer open
)

// Step is a step of the catalog: the permission that resolves it, the
// module it needs (empty for the core) and whether it is critical. A
// critical step brings back a hidden checklist when it goes missing.
type Step struct {
	ID         StepID
	Permission string
	Module     string
	Critical   bool
	// applies is whether the step concerns the tenant at all (nil: always).
	applies func(Facts) bool
	// done is whether the tenant's data already resolves it.
	done func(Facts) bool
}

// catalog is every step. The invoicing step applies only once electronic
// invoicing is turned on: until its screen exists (T36c) nobody could
// resolve it from the app.
var catalog = []Step{
	{ID: StepBusinessProfile, Permission: "tenant:manage", done: func(f Facts) bool { return f.HasContact }},
	{
		ID: StepInvoicing, Permission: "invoicing:manage", Critical: true,
		applies: func(f Facts) bool { return f.InvoicingEnabled },
		done:    func(f Facts) bool { return f.InvoicingEnabled && f.InvoicingReady },
	},
	{ID: StepProducts, Permission: "catalog:write", done: func(f Facts) bool { return f.HasProduct }},
	{ID: StepAlternateUnits, Permission: "catalog:write", Module: "unit_conversions", done: func(f Facts) bool { return f.HasAlternateUnit }},
	{ID: StepVariantProducts, Permission: "catalog:write", Module: "variants", done: func(f Facts) bool { return f.HasVariantProduct }},
	{ID: StepInitialStock, Permission: "inventory:adjust", done: func(f Facts) bool { return f.HasStock }},
	{ID: StepReorderRules, Permission: "inventory:adjust", done: func(f Facts) bool { return f.HasReorderRule }},
	{ID: StepCustomerCredit, Permission: "receivables:manage", done: func(f Facts) bool { return f.HasCreditCustomer }},
	{ID: StepInviteTeam, Permission: "users:manage", done: func(f Facts) bool { return f.HasTeammate }},
	{ID: StepFirstSale, Permission: "sales:create", done: func(f Facts) bool { return f.HasSale }},
}

// Steps returns the catalog.
func Steps() []Step { return slices.Clone(catalog) }

// stepByID returns the catalog's step id.
func stepByID(id StepID) (Step, bool) {
	i := slices.IndexFunc(catalog, func(s Step) bool { return s.ID == id })
	if i < 0 {
		return Step{}, false
	}
	return catalog[i], true
}

// ValidateSteps checks a template's steps: at least one, all from the
// catalog and none repeated. It runs at startup with every template.
func ValidateSteps(ids []string) error {
	if len(ids) == 0 {
		return ErrNoSteps
	}
	for i, id := range ids {
		if _, ok := stepByID(StepID(id)); !ok {
			return fmt.Errorf("%w: %q", ErrUnknownStep, id)
		}
		if slices.Contains(ids[:i], id) {
			return fmt.Errorf("%w: %q", ErrRepeatedStep, id)
		}
	}
	return nil
}
