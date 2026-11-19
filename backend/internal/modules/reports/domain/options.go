package domain

import "slices"

// SalesGroup is how the sales report groups the sales.
type SalesGroup string

// The sales report's groupings.
const (
	SalesByDay           SalesGroup = "day"
	SalesByUser          SalesGroup = "user"
	SalesByPaymentMethod SalesGroup = "payment_method"
	SalesByCategory      SalesGroup = "category"
	SalesByLocation      SalesGroup = "location"
)

// CashGroup is how the cash closings report groups the closings.
type CashGroup string

// The cash closings report's groupings.
const (
	CashByUser     CashGroup = "user"
	CashByLocation CashGroup = "location"
	CashByDay      CashGroup = "day"
)

// AdjustmentGroup is how the stock adjustments report groups them.
type AdjustmentGroup string

// The stock adjustments report's groupings.
const (
	AdjustmentsByUser    AdjustmentGroup = "user"
	AdjustmentsByVariant AdjustmentGroup = "variant"
)

// ProductSort orders the products report.
type ProductSort string

// The products report's orders, most first.
const (
	ByQuantity ProductSort = "quantity"
	ByRevenue  ProductSort = "revenue"
	ByProfit   ProductSort = "profit"
)

// Bucket is how long each point of a trend is: a day, a week from Monday
// or a calendar month.
type Bucket string

// The trends' buckets.
const (
	BucketDay   Bucket = "day"
	BucketWeek  Bucket = "week"
	BucketMonth Bucket = "month"
)

// option returns v if it is one of valid, the default (valid[0]) if it is
// empty, or ErrInvalidFilter.
func option[T ~string](v T, valid ...T) (T, error) {
	if v == "" {
		return valid[0], nil
	}
	if !slices.Contains(valid, v) {
		return "", ErrInvalidFilter
	}
	return v, nil
}

// ParseSalesGroup checks g; empty is by day.
func ParseSalesGroup(g string) (SalesGroup, error) {
	return option(SalesGroup(g), SalesByDay, SalesByUser, SalesByPaymentMethod, SalesByCategory, SalesByLocation)
}

// ParseCashGroup checks g; empty is by user.
func ParseCashGroup(g string) (CashGroup, error) {
	return option(CashGroup(g), CashByUser, CashByLocation, CashByDay)
}

// ParseAdjustmentGroup checks g; empty is by user.
func ParseAdjustmentGroup(g string) (AdjustmentGroup, error) {
	return option(AdjustmentGroup(g), AdjustmentsByUser, AdjustmentsByVariant)
}

// ParseProductSort checks s; empty is by revenue.
func ParseProductSort(s string) (ProductSort, error) {
	return option(ProductSort(s), ByRevenue, ByQuantity, ByProfit)
}

// ParseBucket checks b; empty is by week.
func ParseBucket(b string) (Bucket, error) {
	return option(Bucket(b), BucketWeek, BucketDay, BucketMonth)
}
