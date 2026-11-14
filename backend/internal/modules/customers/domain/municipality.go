package domain

import "context"

// Municipality is a municipality of the DANE's DIVIPOLA catalog: its code
// ("05001") and its department's ("05"). The catalog is the same for every
// tenant.
type Municipality struct {
	Code, Name                     string
	DepartmentCode, DepartmentName string
}

// MunicipalityCatalog reads the DANE's catalog of municipalities.
type MunicipalityCatalog interface {
	// List returns every municipality, by department and name.
	List(ctx context.Context) ([]Municipality, error)
}
