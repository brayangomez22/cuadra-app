// Package domain holds the inventory module's entities: Location (a store or
// warehouse), StockLevel (a product's stock at a location) and the immutable
// StockMovement that records every change in the kardex. It depends only on
// the standard library, uuid and decimal.
package domain
