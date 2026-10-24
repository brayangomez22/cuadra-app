package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// PriceQuery is what a price rule knows of a line of the cart: who sells,
// where and to whom, the variant, the unit it is sold in, its own price in
// that unit (BasePrice, without IVA) and how many of that unit the line
// takes. The price a rule gives is per unit of the line too.
type PriceQuery struct {
	TenantID   uuid.UUID
	LocationID uuid.UUID
	CustomerID *uuid.UUID
	VariantID  uuid.UUID
	ProductID  uuid.UUID
	BasePrice  decimal.Decimal
	Quantity   decimal.Decimal
	// UnitID is the alternate unit of the line, or nil for the base unit.
	UnitID *uuid.UUID
	// UnitFactor is how many base units one unit of the line holds: 1 for
	// the base unit.
	UnitFactor decimal.Decimal
}

// ResolvedPrice is a line's unit price without IVA and the rule that gave
// it. The price is not rounded: the sale line rounds it.
type ResolvedPrice struct {
	UnitPrice decimal.Decimal
	Rule      domain.PriceRuleID
}

// PriceResolver gives the unit price of a line of the cart. ConfirmSale
// calls it for each line before the transaction that stores the sale.
type PriceResolver interface {
	Resolve(ctx context.Context, q PriceQuery) (ResolvedPrice, error)
}

// PriceRule is one rule of a PriceChain, which a module (price lists,
// volume pricing, promotions...) provides and internal/server registers. Apply
// returns the unit price without IVA and true, or false to let the next rule
// decide.
type PriceRule interface {
	ID() domain.PriceRuleID
	Apply(ctx context.Context, q PriceQuery) (decimal.Decimal, bool, error)
}

// PriceChain is the PriceResolver of the core: its rules in the order they
// were registered and, last, the variant's own price (variant_price). The
// first rule that applies gives the price.
type PriceChain struct {
	rules []PriceRule
}

// NewPriceChain builds a chain with rules before the base rule. Each rule
// needs a valid id, distinct from the others and from variant_price.
func NewPriceChain(rules ...PriceRule) (*PriceChain, error) {
	seen := map[domain.PriceRuleID]bool{domain.VariantPriceRule: true}
	for _, r := range rules {
		if r == nil {
			return nil, errors.New("sales: nil price rule")
		}
		id, err := domain.NewPriceRuleID(string(r.ID()))
		if err != nil {
			return nil, fmt.Errorf("sales: price rule %q: %w", r.ID(), err)
		}
		if seen[id] {
			return nil, fmt.Errorf("sales: price rule %q registered twice", id)
		}
		seen[id] = true
	}
	return &PriceChain{rules: rules}, nil
}

// Resolve returns the price of the first rule that applies, or the
// variant's price.
func (c *PriceChain) Resolve(ctx context.Context, q PriceQuery) (ResolvedPrice, error) {
	for _, r := range c.rules {
		price, ok, err := r.Apply(ctx, q)
		if err != nil {
			return ResolvedPrice{}, fmt.Errorf("sales: price rule %s: %w", r.ID(), err)
		}
		if ok {
			return ResolvedPrice{UnitPrice: price, Rule: r.ID()}, nil
		}
	}
	return ResolvedPrice{UnitPrice: q.BasePrice, Rule: domain.VariantPriceRule}, nil
}
