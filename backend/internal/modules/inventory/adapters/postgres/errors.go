package postgres

// Constraints of migrations/00005_inventory.sql and 00012_stock_by_variant.sql
// the adapter maps to domain errors.
const (
	locationsNameKey        = "locations_tenant_id_name_key"
	stockLevelsVariantFKey  = "stock_levels_variant_fkey"
	stockLevelsLocationFKey = "stock_levels_location_fkey"
)
