package domain

import "github.com/google/uuid"

// Scope is the tenant and, optionally, the location a report reads.
type Scope struct {
	TenantID   uuid.UUID
	LocationID *uuid.UUID
}

// ProductScope narrows a products report to a category and to the products
// whose tenant attribute Attribute (its key) is AttributeValue.
type ProductScope struct {
	CategoryID     *uuid.UUID
	Attribute      string
	AttributeValue string
}

// NewProductScope checks that an attribute filter has both its key and its
// value, or neither.
func NewProductScope(categoryID *uuid.UUID, attribute, value *string) (ProductScope, error) {
	s := ProductScope{CategoryID: categoryID}
	if (attribute == nil) != (value == nil) {
		return ProductScope{}, ErrInvalidFilter
	}
	if attribute != nil {
		s.Attribute, s.AttributeValue = *attribute, *value
	}
	return s, nil
}

// Page is a page of rows.
type Page struct{ Limit, Offset int }

// SalesFilter is what the sales report reads.
type SalesFilter struct {
	Scope
	Period Period
	Group  SalesGroup
}

// ProductFilter is what the products report reads.
type ProductFilter struct {
	Scope
	ProductScope
	Period Period
}

// SaleProfitFilter is what the profit by sale report reads.
type SaleProfitFilter struct {
	Scope
	Period Period
	Page
}

// IdleFilter is what the idle stock report reads: variants with stock not
// sold since the start of Since's day.
type IdleFilter struct {
	Scope
	ProductScope
	Since Since
}

// CashFilter is what the cash closings report reads.
type CashFilter struct {
	Scope
	Period Period
}

// TeamFilter is what the team control report reads.
type TeamFilter struct {
	Scope
	Period Period
	Bucket Bucket
}

// AdjustmentFilter is what the stock adjustments report reads.
type AdjustmentFilter struct {
	Scope
	Period Period
	Group  AdjustmentGroup
}
