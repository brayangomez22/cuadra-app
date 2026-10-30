package demo

import (
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/server"
)

// HistoryDays is how long the history of a full demo is: six months.
const HistoryDays = 182

// DefaultSeed is the seed of the public demo: the E2E and load tests can
// count on its data.
const DefaultSeed = 1

// bogota is where the business's hours are. Colombia has no daylight saving
// time.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// taxRate is the IVA of every demo article: the default of both templates.
var taxRate = decimal.RequireFromString("0.19")

// Options choose the scenario: the same options give the same scenario.
// Zero values take the profile's (or HistoryDays); tests ask for a small
// one.
type Options struct {
	Seed uint64
	// End is the last day of history, as a date (its time is ignored).
	End         time.Time
	Days        int
	Products    int
	SalesPerDay int
	Customers   int
}

// Scenario is everything a demo tenant does, in order: its catalog,
// customers and suppliers, and day by day its purchases, sales and
// payments. It is pure data: Run writes it through the API.
type Scenario struct {
	Profile   Profile
	Locations []string
	Products  []Product
	Customers []Customer
	Suppliers []Supplier
	Days      []Day
	// Reorder are the minimum and maximum stock of some variants.
	Reorder []ReorderRule
}

// ReorderRule is the minimum and maximum stock of a variant at a location.
type ReorderRule struct {
	Location, Product, Variant int
	Min, Max                   decimal.Decimal
}

// Product is an article of the catalog with its variants.
type Product struct {
	Name       string
	Category   string
	Unit       string
	Attributes map[string]string
	Variants   []Variant
	// Supplier is the index of who sells it; Step and MaxQuantity, how a
	// sale line takes it.
	Supplier    int
	Step        decimal.Decimal
	MaxQuantity int
	// PurchaseUnit and PurchaseFactor are the unit it is bought in and how
	// many of Unit it brings; empty and zero when it is bought in Unit.
	PurchaseUnit   string
	PurchaseFactor decimal.Decimal
}

// Variant is what is sold and counted. Axes is empty for a simple product.
type Variant struct {
	SKU     string
	Barcode string
	Axes    map[string]string
	Cost    decimal.Decimal
	Price   decimal.Decimal
}

// Customer is a buyer; a positive CreditLimit gives them credit.
type Customer struct {
	Name         string
	DocumentType string
	Document     string
	PersonType   string
	Phone        string
	City         string
	Segment      string
	CreditLimit  decimal.Decimal
	TermDays     int
	// Defaulter never pays: their debts show up as overdue.
	Defaulter bool
}

// Day is one day of history. Date is the calendar day at midnight UTC.
type Day struct {
	Date      time.Time
	Purchases []Purchase
	Sales     []Sale
	Payments  []Payment
	// Opening and Closing are when the drawers open and close; the
	// purchases arrive before, the payments and sales come in between.
	Opening, Closing time.Time
	// CashDifference is, per location, how much the closing count is off
	// what the drawer should have: zero most days, a cashier's shortfall
	// (negative) or overage in whole thousands on some.
	CashDifference []decimal.Decimal
}

// Sale is a sale at a location; Customer is -1 for an anonymous buyer.
type Sale struct {
	At       time.Time
	Location int
	Customer int
	Lines    []Line
	Method   string
	Credit   bool
}

// Line is a sale or purchase line of a product's variant. Quantity is in
// the product's unit; InPurchaseUnit says a purchase line was bought in the
// product's purchase unit (Quantity is then a multiple of its factor).
type Line struct {
	Product        int
	Variant        int
	Quantity       decimal.Decimal
	InPurchaseUnit bool
}

// Purchase is a supplier invoice received at a location.
type Purchase struct {
	At       time.Time
	Location int
	Supplier int
	Invoice  string
	Credit   bool
	Lines    []Line
}

// Payment is a customer paying part of their debt at the first location.
type Payment struct {
	At       time.Time
	Customer int
	Amount   decimal.Decimal
	Method   string
}

// Event is one thing that happens in a day: exactly one field is set.
type Event struct {
	At       time.Time
	Purchase *Purchase
	Sale     *Sale
	Payment  *Payment
}

// Events returns the day's purchases, sales and payments in time order.
func (d *Day) Events() []Event {
	out := make([]Event, 0, len(d.Purchases)+len(d.Sales)+len(d.Payments))
	for i := range d.Purchases {
		out = append(out, Event{At: d.Purchases[i].At, Purchase: &d.Purchases[i]})
	}
	for i := range d.Sales {
		out = append(out, Event{At: d.Sales[i].At, Sale: &d.Sales[i]})
	}
	for i := range d.Payments {
		out = append(out, Event{At: d.Payments[i].At, Payment: &d.Payments[i]})
	}
	slices.SortStableFunc(out, func(a, b Event) int { return a.At.Compare(b.At) })
	return out
}

// Total is what the sale charges with IVA, by the rule of the sales
// module's lines (gross rounded to cents, then its IVA rounded): the payments
// are for this amount, and credit stays under the limit with it.
func (s Scenario) Total(sale Sale) decimal.Decimal {
	total := decimal.Zero
	for _, l := range sale.Lines {
		gross := s.Products[l.Product].Variants[l.Variant].Price.Mul(l.Quantity).Round(2)
		total = total.Add(gross).Add(gross.Mul(taxRate).Round(2))
	}
	return total
}

// Staff are the demo tenant's users besides its owner, one per role.
func (p Profile) Staff() []server.DemoUser {
	return []server.DemoUser{
		{Name: "Administrador Demo", Email: "admin@" + p.Tenant.Mailbox, Role: server.DemoRoleAdmin},
		{Name: "Cajero Demo", Email: "cajero@" + p.Tenant.Mailbox, Role: server.DemoRoleCashier},
		{Name: "Bodeguero Demo", Email: "bodega@" + p.Tenant.Mailbox, Role: server.DemoRoleWarehouse},
	}
}

// OwnerEmail is the email of the demo tenant's owner.
func (p Profile) OwnerEmail() string { return "dueno@" + p.Tenant.Mailbox }

// NewScenario builds the scenario of profile with options: the same inputs
// give the same scenario.
func NewScenario(p Profile, o Options) (Scenario, error) {
	o = o.withDefaults(p)
	if o.End.IsZero() {
		return Scenario{}, errors.New("demo: options need an end day")
	}
	// #nosec G404 -- demo data must be repeatable, not unpredictable.
	rng := rand.New(rand.NewPCG(o.Seed, o.Seed^0x9e3779b97f4a7c15))
	products, err := planCatalog(rng, p, o.Products)
	if err != nil {
		return Scenario{}, err
	}
	s := Scenario{
		Profile:   p,
		Locations: p.Locations,
		Products:  products,
		Customers: planCustomers(rng, p, o.Customers),
		Suppliers: p.Suppliers,
	}
	h := newHistory(rng, &s, o)
	s.Days = h.plan()
	s.Reorder = h.reorderRules()
	return s, nil
}

// withDefaults fills the zero options from the profile.
func (o Options) withDefaults(p Profile) Options {
	if o.Days <= 0 {
		o.Days = HistoryDays
	}
	if o.Products <= 0 {
		o.Products = p.Products
	}
	if o.SalesPerDay <= 0 {
		o.SalesPerDay = p.SalesPerDay
	}
	if o.Customers <= 0 {
		o.Customers = p.Customers
	}
	o.End = time.Date(o.End.Year(), o.End.Month(), o.End.Day(), 0, 0, 0, 0, time.UTC)
	return o
}
