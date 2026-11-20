package domain

// Facts are what the tenant's data says, read when the checklist is asked
// for. Each step is done when its fact holds.
type Facts struct {
	HasContact        bool // the business has its address, city and phone
	InvoicingEnabled  bool // electronic invoicing is turned on
	InvoicingReady    bool // and a numbering resolution of the default kind is in force
	HasProduct        bool
	HasStock          bool // at least one stock movement
	HasAlternateUnit  bool // a variant sold in another unit
	HasVariantProduct bool // a product with more than one variant
	HasReorderRule    bool
	HasCreditCustomer bool // a customer with a credit limit
	HasTeammate       bool // another user, or an invitation in force
	HasSale           bool // a confirmed sale, even if voided afterwards
}
