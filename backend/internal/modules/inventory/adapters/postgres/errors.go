package postgres

// Constraints of migrations/00005_inventory.sql, 00012_stock_by_variant.sql
// and 00021_reorder_rules.sql the adapter maps to domain errors.
const (
	locationsNameKey         = "locations_tenant_id_name_key"
	stockLevelsVariantFKey   = "stock_levels_variant_fkey"
	stockLevelsLocationFKey  = "stock_levels_location_fkey"
	reorderRulesVariantFKey  = "reorder_rules_variant_fkey"
	reorderRulesLocationFKey = "reorder_rules_location_fkey"
)
