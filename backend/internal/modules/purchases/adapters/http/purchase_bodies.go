package httpadapter

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http/purchasesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func purchaseBody(p *domain.Purchase) purchasesapi.Purchase {
	t := p.Totals()
	body := purchasesapi.Purchase{
		Id: p.ID(), SupplierId: p.SupplierID(), LocationId: p.LocationID(), InvoiceNumber: p.InvoiceNumber(),
		IssuedOn: date(p.IssuedOn()), Payment: purchasesapi.PurchasePayment(p.Payment()),
		Status: purchasesapi.PurchaseStatus(p.Status()), Notes: optional(p.Notes()),
		Subtotal: httpx.FormatMoney(t.Subtotal), Tax: httpx.FormatMoney(t.Tax), Total: httpx.FormatMoney(t.Total),
		BalanceDue: httpx.FormatMoney(p.BalanceDue()), CreatedBy: p.CreatedBy(), CreatedAt: p.CreatedAt(),
		Lines: make([]purchasesapi.PurchaseLine, 0, len(p.Lines())),
		Taxes: make([]purchasesapi.PurchaseTax, 0, len(t.Taxes)),
	}
	if due := p.DueOn(); due != nil {
		body.DueOn = new(date(*due))
	}
	for _, l := range p.Lines() {
		body.Lines = append(body.Lines, lineBody(l))
	}
	for _, x := range t.Taxes {
		body.Taxes = append(body.Taxes, purchasesapi.PurchaseTax{
			Rate: httpx.FormatDecimal(x.Rate), Base: httpx.FormatMoney(x.Base), Tax: httpx.FormatMoney(x.Tax),
		})
	}
	return body
}

func lineBody(l domain.Line) purchasesapi.PurchaseLine {
	u := l.Unit()
	return purchasesapi.PurchaseLine{
		VariantId: l.VariantID(), ProductName: l.ProductName(), VariantLabel: optional(l.VariantLabel()),
		UnitId: u.ID, UnitCode: u.Code, UnitFactor: httpx.FormatDecimal(u.Factor),
		Quantity: httpx.FormatDecimal(l.Quantity()), UnitCost: httpx.FormatMoney(l.UnitCost()),
		TaxRate: httpx.FormatDecimal(l.TaxRate()), Subtotal: httpx.FormatMoney(l.Subtotal()),
		Tax: httpx.FormatMoney(l.Tax()), Total: httpx.FormatMoney(l.Total()),
		BaseQuantity: httpx.FormatDecimal(l.BaseQuantity()), BaseUnitCost: httpx.FormatMoney(l.BaseUnitCost()),
	}
}

func summaryBody(s domain.PurchaseSummary) purchasesapi.PurchaseSummary {
	body := purchasesapi.PurchaseSummary{
		Id: s.ID, SupplierId: s.SupplierID, SupplierName: s.SupplierName, LocationId: s.LocationID,
		InvoiceNumber: s.InvoiceNumber, IssuedOn: date(s.IssuedOn), Payment: purchasesapi.PurchasePayment(s.Payment),
		Status: purchasesapi.PurchaseStatus(s.Status()), Lines: s.Lines, Total: httpx.FormatMoney(s.Total),
		BalanceDue: httpx.FormatMoney(s.BalanceDue), CreatedAt: s.CreatedAt,
	}
	if s.DueOn != nil {
		body.DueOn = new(date(*s.DueOn))
	}
	return body
}
