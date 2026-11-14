package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing"
	invoicingapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	invoicingdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	salesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// SalesInvoicing satisfies salesapp.ElectronicDocuments with the invoicing
// module, translating its errors to the sales module's. The invoicing
// module reads the sales module's sales, so Invoicing is set once both
// exist.
type SalesInvoicing struct{ Invoicing *invoicing.Module }

// RegisterSale numbers the document of a sale being confirmed, in its
// transaction.
func (s *SalesInvoicing) RegisterSale(ctx context.Context, req salesapp.DocumentRequest) error {
	err := s.Invoicing.RegisterSale(ctx, req.TenantID, req.UserID, invoicing.SaleDocumentRequest{
		SaleID: req.SaleID, Kind: req.Kind, Credit: req.Credit, HasCustomer: req.HasCustomer,
	})
	switch {
	case errors.Is(err, invoicingdomain.ErrNoActiveResolution):
		return salesdomain.ErrNoActiveResolution
	case errors.Is(err, invoicingdomain.ErrInvoiceRequiresCustomer):
		return salesdomain.ErrInvoiceRequiresCustomer
	case errors.Is(err, invoicingdomain.ErrInvalidKind):
		return salesdomain.ErrInvalidDocumentKind
	}
	return err
}

// HasDocument is whether the sale has an electronic document.
func (s *SalesInvoicing) HasDocument(ctx context.Context, tenantID, userID, saleID uuid.UUID) (bool, error) {
	return s.Invoicing.HasDocument(ctx, tenantID, userID, saleID)
}

// InvoicingSales satisfies invoicingapp.SaleSource with the sales module.
type InvoicingSales struct{ Sales *sales.Module }

// Sale reads what a document says of its sale.
func (s InvoicingSales) Sale(ctx context.Context, tenantID, userID, saleID uuid.UUID) (invoicingapp.SaleData, error) {
	sale, err := s.Sales.Sale(ctx, tenantID, userID, saleID)
	if err != nil {
		return invoicingapp.SaleData{}, err
	}
	totals := sale.Totals()
	data := invoicingapp.SaleData{
		Number: sale.Number(), LocationID: sale.LocationID(), CustomerID: sale.CustomerID(),
		Subtotal: totals.Subtotal, Discounts: totals.Discounts, Tax: totals.Tax, Total: totals.Total,
	}
	for _, l := range sale.Lines() {
		data.Lines = append(data.Lines, invoicingapp.SaleLine{
			VariantID: l.VariantID(), Name: lineName(l), Quantity: l.Quantity(), UnitPrice: l.UnitPrice(),
			Discount: l.Discount(), TaxRate: l.TaxRate(), Tax: l.Tax(), Total: l.Total(),
		})
	}
	for _, p := range sale.Payments() {
		data.Payments = append(data.Payments, invoicingapp.SalePayment{Method: string(p.Method()), Amount: p.Amount()})
	}
	return data, nil
}

// lineName is the product's name with its variant's label, if it has one.
func lineName(l salesdomain.SaleLine) string {
	if l.VariantLabel() == "" {
		return l.ProductName()
	}
	return l.ProductName() + " " + l.VariantLabel()
}

// InvoicingBuyers satisfies invoicingapp.BuyerDirectory with the customers
// module.
type InvoicingBuyers struct{ Customers *customers.Module }

// Buyer reads who a customer is, for the document.
func (b InvoicingBuyers) Buyer(ctx context.Context, tenantID, userID, customerID uuid.UUID) (invoicingdomain.Buyer, error) {
	c, err := b.Customers.Customer(ctx, tenantID, userID, customerID)
	if err != nil {
		return invoicingdomain.Buyer{}, err
	}
	id := c.ID()
	return invoicingdomain.Buyer{
		CustomerID: &id, PersonType: string(c.PersonType()), DocumentType: string(c.Document().Type()),
		DocumentNumber: c.Document().Number(), Name: c.Name(), Email: c.Contact().Email,
	}, nil
}
