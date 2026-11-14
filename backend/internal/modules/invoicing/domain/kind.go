package domain

// Kind is the type of electronic document a sale is issued as, by the codes
// of the contract. The provider's adapter maps it to the DIAN code.
type Kind string

// The kinds of document of a sale.
const (
	// KindPOS is the electronic equivalent document of a point of sale
	// (DE POS, DIAN code 20): the counter sale.
	KindPOS Kind = "pos"
	// KindInvoice is the electronic sales invoice (FE, DIAN code 01).
	KindInvoice Kind = "invoice"
)

// ParseKind returns the Kind of s, or ErrInvalidKind.
func ParseKind(s string) (Kind, error) {
	switch k := Kind(s); k {
	case KindPOS, KindInvoice:
		return k, nil
	}
	return "", ErrInvalidKind
}

// KindChoice is what decides the kind of a sale's document.
type KindChoice struct {
	// Requested is the kind the sale asked for, or "" to let the rules
	// decide.
	Requested Kind
	// Credit is whether part of the sale is paid on credit.
	Credit bool
	// HasCustomer is whether the sale names a customer (otherwise it is to
	// the final consumer).
	HasCustomer bool
	// Default is the tenant's kind for the sales that ask for none.
	Default Kind
}

// ChooseKind decides the kind of a sale's document: a credit sale is always
// invoiced; otherwise the requested kind, or the tenant's default. An
// invoice the sale asks for needs a customer (ErrInvoiceRequiresCustomer);
// the tenant's default of invoicing everything goes to the final consumer
// when there is none, as the DIAN allows.
func ChooseKind(c KindChoice) (Kind, error) {
	if c.Requested != "" {
		if _, err := ParseKind(string(c.Requested)); err != nil {
			return "", err
		}
	}
	switch {
	case c.Credit:
		return KindInvoice, nil
	case c.Requested == KindInvoice && !c.HasCustomer:
		return "", ErrInvoiceRequiresCustomer
	case c.Requested != "":
		return c.Requested, nil
	}
	return c.Default, nil
}
