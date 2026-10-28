package postgres

// Constraints of migrations/00005_inventory.sql, 00012_stock_by_variant.sql,
// 00021_reorder_rules.sql and 00023_stock_counts.sql the adapter maps to
// domain errors.
const (
	locationsNameKey         = "locations_tenant_id_name_key"
	stockLevelsVariantFKey   = "stock_levels_variant_fkey"
	stockLevelsLocationFKey  = "stock_levels_location_fkey"
	reorderRulesVariantFKey  = "reorder_rules_variant_fkey"
	reorderRulesLocationFKey = "reorder_rules_location_fkey"
	stockCountsOneOpen       = "stock_counts_one_open"
	stockCountsLocationFKey  = "stock_counts_location_fkey"
	countLinesVariantFKey    = "stock_count_lines_variant_fkey"
)
