package demo

import (
	"context"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// The drawers: each location's opens at openingHour with openingFloat and
// closes at closingMinute, counted in these bills and coins.
const closingMinute = 19*60 + 30

var (
	openingFloat  = decimal.NewFromInt(200_000)
	denominations = []int64{100_000, 50_000, 20_000, 10_000, 5_000, 2_000, 1_000, 500, 200, 100, 50}
)

// expectedCash is what a drawer should have at closing.
type expectedCash = decimal.Decimal

// seller is who sells at a location: the cashier at the first, the
// administrator at the second, the owner at any other.
func (r *runner) seller(loc int) string {
	staff := r.s.Profile.Staff()
	switch loc {
	case 0:
		return staff[1].Email
	case 1:
		return staff[0].Email
	}
	return r.s.Profile.OwnerEmail()
}

// writeDay writes the day's events in time order: the purchases before
// opening, then the drawers open, the payments and the sales, and the
// drawers close.
func (r *runner) writeDay(ctx context.Context, d *Day) error {
	opening := at(d.Date, openingHour*60)
	open := false
	for _, e := range d.Events() {
		if !open && !e.At.Before(opening) {
			if err := r.openDrawers(ctx, d); err != nil {
				return err
			}
			open = true
		}
		r.clock.Set(e.At)
		if err := r.write(ctx, d, e); err != nil {
			return err
		}
	}
	if !open {
		if err := r.openDrawers(ctx, d); err != nil {
			return err
		}
	}
	return r.closeDrawers(ctx, d)
}

// write sends one event to the API.
func (r *runner) write(ctx context.Context, d *Day, e Event) error {
	switch {
	case e.Purchase != nil:
		return r.purchase(ctx, d, *e.Purchase)
	case e.Payment != nil:
		return r.payment(ctx, *e.Payment)
	default:
		return r.sale(ctx, *e.Sale)
	}
}

// openDrawers opens every location's drawer with its seller.
func (r *runner) openDrawers(ctx context.Context, d *Day) error {
	r.clock.Set(at(d.Date, openingHour*60))
	r.sessions = r.sessions[:0]
	r.expected = r.expected[:0]
	for loc, id := range r.ids.locations {
		res, err := r.api.OpenCashSessionWithResponse(ctx, apiclient.OpenCashSessionInput{
			LocationId: id, OpeningFloat: httpx.FormatMoney(openingFloat),
		}, r.as(r.seller(loc)))
		if err := expect("open drawer", res, err); err != nil {
			return err
		}
		r.sessions = append(r.sessions, res.JSON201.Id)
		r.expected = append(r.expected, openingFloat)
	}
	return nil
}

// closeDrawers counts each drawer as it should be, down to the smallest
// coin: what is under 50 pesos stays as a small difference.
func (r *runner) closeDrawers(ctx context.Context, d *Day) error {
	r.clock.Set(at(d.Date, closingMinute))
	for loc, id := range r.sessions {
		res, err := r.api.CloseCashSessionWithResponse(ctx, id, apiclient.CloseCashSessionInput{
			Counts: count(r.expected[loc]),
		}, r.as(r.seller(loc)))
		if err := expect("close drawer", res, err); err != nil {
			return err
		}
	}
	return nil
}

// count splits amount in the fewest bills and coins.
func count(amount decimal.Decimal) []apiclient.DenominationCount {
	left := amount.IntPart()
	var out []apiclient.DenominationCount
	for _, d := range denominations {
		if n := left / d; n > 0 {
			out = append(out, apiclient.DenominationCount{Denomination: httpx.FormatMoney(decimal.NewFromInt(d)), Quantity: n})
			left -= n * d
		}
	}
	return out
}

// sale confirms a sale with what pays it: its total on credit or by card
// and transfers, or rounded up to a thousand in cash (with change).
func (r *runner) sale(ctx context.Context, s Sale) error {
	total := r.s.Total(s)
	paid := total
	if s.Method == "cash" {
		paid = total.Div(decimal.NewFromInt(1000)).Ceil().Mul(decimal.NewFromInt(1000))
		r.expected[s.Location] = r.expected[s.Location].Add(total)
	}
	in := apiclient.ConfirmSaleInput{
		LocationId: r.ids.locations[s.Location],
		Payments:   []apiclient.PaymentInput{{Method: apiclient.PaymentMethod(s.Method), Amount: httpx.FormatMoney(paid)}},
	}
	if s.Customer >= 0 {
		in.CustomerId = &r.ids.customers[s.Customer]
	}
	for _, l := range s.Lines {
		in.Lines = append(in.Lines, apiclient.SaleLineInput{
			VariantId: r.ids.variants[l.Product][l.Variant], Quantity: httpx.FormatDecimal(l.Quantity),
		})
	}
	res, err := r.api.ConfirmSaleWithResponse(ctx, in, r.as(r.seller(s.Location)))
	return expect("confirm sale", res, err)
}

// purchase registers a supplier's invoice at its cost, due by the
// supplier's term when it is on credit.
func (r *runner) purchase(ctx context.Context, d *Day, p Purchase) error {
	tax := httpx.FormatDecimal(taxRate)
	in := apiclient.PurchaseInput{
		SupplierId: r.ids.suppliers[p.Supplier], LocationId: r.ids.locations[p.Location],
		InvoiceNumber: p.Invoice, IssuedOn: openapi_types.Date{Time: d.Date}, Payment: apiclient.PurchasePaymentCash,
	}
	if p.Credit {
		in.Payment = apiclient.PurchasePaymentCredit
		in.DueOn = &openapi_types.Date{Time: d.Date.AddDate(0, 0, r.s.Suppliers[p.Supplier].TermDays)}
	}
	for _, l := range p.Lines {
		in.Lines = append(in.Lines, r.purchaseLine(l, &tax))
	}
	res, err := r.api.RegisterPurchaseWithResponse(ctx, in, r.as(r.s.Profile.Staff()[0].Email))
	return expect(fmt.Sprintf("register purchase %s", p.Invoice), res, err)
}

// purchaseLine is a line at the variant's cost, in its purchase unit (so
// many boxes, at the cost of a box) when it was bought in one.
func (r *runner) purchaseLine(l Line, tax *apiclient.TaxRate) apiclient.PurchaseLineInput {
	product := r.s.Products[l.Product]
	quantity, cost := l.Quantity, product.Variants[l.Variant].Cost
	in := apiclient.PurchaseLineInput{VariantId: r.ids.variants[l.Product][l.Variant], TaxRate: tax}
	if l.InPurchaseUnit {
		unit := r.ids.units[product.PurchaseUnit]
		in.UnitId = &unit
		quantity, cost = quantity.Div(product.PurchaseFactor), cost.Mul(product.PurchaseFactor)
	}
	in.Quantity, in.UnitCost = httpx.FormatDecimal(quantity), httpx.FormatMoney(cost)
	return in
}

// payment receives a customer's payment at the first location's drawer.
func (r *runner) payment(ctx context.Context, p Payment) error {
	if p.Method == "cash" {
		r.expected[0] = r.expected[0].Add(p.Amount)
	}
	res, err := r.api.RecordCustomerPaymentWithResponse(ctx, r.ids.customers[p.Customer], apiclient.CustomerPaymentInput{
		Amount: httpx.FormatMoney(p.Amount), Method: apiclient.CustomerPaymentMethod(p.Method),
	}, r.as(r.seller(0)))
	return expect("record payment", res, err)
}
