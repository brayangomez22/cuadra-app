package postgres

// Constraints of migrations/00004_catalog.sql, 00010_product_variants.sql,
// 00014_units_of_measure.sql, 00015_product_attributes.sql,
// 00016_variant_axes.sql and 00020_variant_units.sql the adapter maps to
// domain errors.
const (
	productsCategoryFKey  = "products_category_fkey"
	productsBaseUnitFKey  = "products_base_unit_fkey"
	unitsCodeKey          = "units_of_measure_tenant_id_code_key"
	categoriesParentFKey  = "categories_parent_fkey"
	variantsSKUKey        = "product_variants_tenant_id_sku_key"
	variantsBarcodeKey    = "product_variants_tenant_id_barcode_key"
	variantsAxisValuesKey = "product_variants_axis_values_key"
	attributesKeyKey      = "attribute_definitions_tenant_id_key_key"
	axesNameKey           = "variant_axes_tenant_id_name_key"
	// 00020_variant_units.sql.
	variantUnitsBarcodeKey = "variant_units_barcode_key"
	variantUnitsUnitFKey   = "variant_units_unit_fkey"
)
