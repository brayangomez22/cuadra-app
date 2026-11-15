package domain

// Issuer is the company that issues the documents: the tenant, as the
// identity module knows it. The provider's adapter sends it as the issuer
// and the branch of each document.
type Issuer struct {
	// NIT is the canonical NIT with its check digit (900123456-1).
	NIT     string
	Name    string
	Address string
	City    string
	// MunicipalityCode is the DANE code of its municipality (05001), or ""
	// if the tenant did not set it.
	MunicipalityCode string
	Phone            string
	// Demo is whether the tenant is a demonstration one, whose documents
	// never reach the provider.
	Demo bool
}
