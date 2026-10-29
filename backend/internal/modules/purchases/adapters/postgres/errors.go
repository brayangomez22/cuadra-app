// Package postgres stores the purchases module's suppliers and purchases in
// PostgreSQL, within the tenant's Row-Level Security (db.WithTenantTx).
package postgres

// Constraints whose violations map to domain errors, by name (00026).
const (
	supplierNITKey       = "suppliers_nit_key"
	purchaseInvoiceKey   = "purchases_invoice_key"
	purchaseSupplierFKey = "purchases_supplier_fkey"
	purchaseLocationFKey = "purchases_location_fkey"
	purchaseVariantFKey  = "purchase_lines_variant_fkey"
	purchaseUnitFKey     = "purchase_lines_unit_fkey"
)

// nullable stores "" as NULL.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
