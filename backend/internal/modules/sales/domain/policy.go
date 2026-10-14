package domain

// SalesPolicy holds the tenant's sales rules. The zero value is the default:
// users without the sales:discount permission cannot give discounts.
type SalesPolicy struct {
	// MaxDiscount is the discount limit of users without sales:discount.
	MaxDiscount DiscountLimit
	// CashSessionOptional lets users sell without an open cash session. The
	// zero value requires one.
	CashSessionOptional bool
}
