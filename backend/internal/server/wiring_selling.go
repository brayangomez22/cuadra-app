package server

import (
	"cmp"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// sellingModules are the modules of a sale: the sale itself, its credit and
// its electronic document.
type sellingModules struct {
	sales       *sales.Module
	receivables *receivables.Module
	invoicing   *invoicing.Module
}

// selling builds the sales, receivables and invoicing modules. Sales charge
// credit sales to the receivables, whose cash payments go into the sales
// module's drawers, and number their documents in the invoicing module,
// which reads the sales: both ports are joined once the modules exist.
func (w *wiring) selling(c *catalog.Module, i *inventory.Module, cu *customers.Module, id *identity.Module) (sellingModules, error) {
	credit, documents := &ports.SalesCredit{}, &ports.SalesInvoicing{}
	s, err := w.sales(c, i, cu, salesCredit{credit, id}, documents)
	if err != nil {
		return sellingModules{}, err
	}
	r, err := w.receivables(cu, s)
	if err != nil {
		return sellingModules{}, err
	}
	inv, err := w.invoicing(s, cu)
	if err != nil {
		return sellingModules{}, err
	}
	credit.Receivables, documents.Invoicing = r, inv
	return sellingModules{sales: s, receivables: r, invoicing: inv}, nil
}

// salesCredit is how sales on credit reach the receivables (set once they
// exist) and who authorizes one over the customer's credit.
type salesCredit struct {
	ledger   *ports.SalesCredit
	identity *identity.Module
}

// sales sells the catalog's variants from the inventory's stock, to the
// customers module's customers, on credit through the receivables, with
// the electronic document of the invoicing module.
func (w *wiring) sales(
	c *catalog.Module, i *inventory.Module, cu *customers.Module, cr salesCredit, docs *ports.SalesInvoicing,
) (*sales.Module, error) {
	return sales.New(sales.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Catalog:        ports.SalesCatalog{Catalog: c},
		Stock:          ports.SalesStock{Inventory: i},
		Customers:      ports.SalesCustomers{Customers: cu},
		Credit:         cr.ledger,
		Authorizer:     ports.SalesAuthorizer{Identity: cr.identity},
		Users:          cr.identity,
		Documents:      docs,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// receivables gives credit to the customers module's customers and takes
// their cash payments into the sales module's drawers.
func (w *wiring) receivables(cu *customers.Module, s *sales.Module) (*receivables.Module, error) {
	return receivables.New(receivables.Config{
		DB:             w.DB,
		Customers:      ports.ReceivablesCustomers{Customers: cu},
		Cash:           ports.ReceivablesCash{Sales: s},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// invoicing issues the electronic documents of the sales module's sales to
// the customers module's customers, in the process's workers.
func (w *wiring) invoicing(s *sales.Module, cu *customers.Module) (*invoicing.Module, error) {
	return invoicing.New(invoicing.Config{
		DB:             w.DB,
		Jobs:           w.Jobs,
		Workers:        w.Workers,
		Schedule:       w.Schedule,
		Provider:       cmp.Or(w.InvoicingProvider, invoicing.ProviderSimulated),
		Sales:          ports.InvoicingSales{Sales: s},
		Buyers:         ports.InvoicingBuyers{Customers: cu},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}
