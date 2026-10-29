package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
)

// pageSize is the contract's largest page.
const pageSize = 100

// Fingerprint summarizes a demo tenant as the API shows it, without ids or
// the real time: the catalog, the stock of each location, the reorder
// alerts, the sales of each day, the purchases and the receivables' aging. The same scenario gives
// the same fingerprint, however many times it is generated.
type Fingerprint struct {
	Products  int
	Sales     int
	Purchases int
	// Alerts are the variants under their minimum stock.
	Alerts int
	Digest string
}

// fingerprint is what a Fingerprint is computed from, one line per fact.
type fingerprint struct {
	r     *runner
	owner apiclient.RequestEditorFn
	lines []string
	skus  map[uuid.UUID]string
	locs  map[uuid.UUID]string
}

// Fingerprint reads the demo tenant of s as its owner on the morning after
// its last day (the aging depends on the date).
func (g Generator) Fingerprint(ctx context.Context, tenant uuid.UUID, s Scenario, password string) (Fingerprint, error) {
	r, err := g.newRunner(&s)
	if err != nil {
		return Fingerprint{}, err
	}
	r.clock.Set(at(s.Days[len(s.Days)-1].Date.AddDate(0, 0, 1), openingHour*60))
	if err := r.logIn(ctx, tenant, password); err != nil {
		return Fingerprint{}, err
	}
	f := &fingerprint{r: r, owner: r.as(s.Profile.OwnerEmail()), skus: map[uuid.UUID]string{}, locs: map[uuid.UUID]string{}}
	out := Fingerprint{}
	steps := []func(context.Context) (int, error){f.products, f.stock, f.sales, f.purchases, f.aging, f.alerts}
	counts := make([]int, len(steps))
	for i, step := range steps {
		if counts[i], err = step(ctx); err != nil {
			return Fingerprint{}, err
		}
	}
	out.Products, out.Sales, out.Purchases, out.Alerts = counts[0], counts[2], counts[3], counts[5]
	slices.Sort(f.lines)
	sum := sha256.Sum256([]byte(strings.Join(f.lines, "\n")))
	out.Digest = hex.EncodeToString(sum[:])
	return out, nil
}

// deref is the value of an optional field, or "-".
func deref[T any](p *T) any {
	if p == nil {
		return "-"
	}
	return *p
}

func (f *fingerprint) add(format string, args ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, args...))
}

// products adds each variant's SKU, barcode, cost and price.
func (f *fingerprint) products(ctx context.Context) (int, error) {
	total := 0
	for offset := 0; ; offset += pageSize {
		limit, off := pageSize, offset
		res, err := f.r.api.ListProductsWithResponse(ctx, &apiclient.ListProductsParams{Limit: &limit, Offset: &off}, f.owner)
		if err := expect("list products", res, err); err != nil {
			return 0, err
		}
		for _, p := range res.JSON200.Items {
			for _, v := range p.Variants {
				f.skus[v.Id] = v.Sku
				f.add("product|%s|%s|%s|%s", p.Name, v.Sku, deref(v.Cost), v.Price)
			}
		}
		total = res.JSON200.Total
		if offset+pageSize >= total {
			return total, nil
		}
	}
}

// stock adds the quantity of each SKU at each location, by location name.
func (f *fingerprint) stock(ctx context.Context) (int, error) {
	locs, err := f.r.api.ListLocationsWithResponse(ctx, f.owner)
	if err := expect("list locations", locs, err); err != nil {
		return 0, err
	}
	for _, loc := range locs.JSON200.Items {
		f.locs[loc.Id] = loc.Name
		if err := f.locationStock(ctx, loc.Id, loc.Name); err != nil {
			return 0, err
		}
	}
	return len(locs.JSON200.Items), nil
}

// locationStock adds the stock of one location.
func (f *fingerprint) locationStock(ctx context.Context, loc uuid.UUID, name string) error {
	for offset := 0; ; offset += pageSize {
		limit, off := pageSize, offset
		res, err := f.r.api.ListStockWithResponse(ctx, loc, &apiclient.ListStockParams{Limit: &limit, Offset: &off}, f.owner)
		if err := expect("list stock", res, err); err != nil {
			return err
		}
		for _, l := range res.JSON200.Items {
			f.add("stock|%s|%s|%s", name, f.skus[l.VariantId], l.Quantity)
		}
		if offset+pageSize >= res.JSON200.Total {
			return nil
		}
	}
}

// sales adds each day's sales: number, time, status and total.
func (f *fingerprint) sales(ctx context.Context) (int, error) {
	n := 0
	for _, d := range f.r.s.Days {
		date := openapi_types.Date{Time: d.Date}
		for offset := 0; ; offset += pageSize {
			limit, off := pageSize, offset
			res, err := f.r.api.ListSalesWithResponse(ctx, &apiclient.ListSalesParams{Date: &date, Limit: &limit, Offset: &off}, f.owner)
			if err := expect("list sales", res, err); err != nil {
				return 0, err
			}
			for _, s := range res.JSON200.Items {
				f.add("sale|%d|%s|%s|%s", s.Number, s.ConfirmedAt.UTC().Format(time.RFC3339), s.Status, s.Total)
			}
			n += len(res.JSON200.Items)
			if offset+pageSize >= res.JSON200.Total {
				break
			}
		}
	}
	return n, nil
}

// purchases adds each purchase's invoice, dates, status, total and balance.
func (f *fingerprint) purchases(ctx context.Context) (int, error) {
	for offset := 0; ; offset += pageSize {
		limit, off := pageSize, offset
		res, err := f.r.api.ListPurchasesWithResponse(ctx, &apiclient.ListPurchasesParams{Limit: &limit, Offset: &off}, f.owner)
		if err := expect("list purchases", res, err); err != nil {
			return 0, err
		}
		for _, p := range res.JSON200.Items {
			f.add("purchase|%s|%s|%s|%s|%s|%s", p.SupplierName, p.InvoiceNumber, p.IssuedOn, p.Status, p.Total, p.BalanceDue)
		}
		if offset+pageSize >= res.JSON200.Total {
			return res.JSON200.Total, nil
		}
	}
}

// aging adds the receivables' totals by age.
func (f *fingerprint) aging(ctx context.Context) (int, error) {
	res, err := f.r.api.GetReceivablesAgingWithResponse(ctx, &apiclient.GetReceivablesAgingParams{}, f.owner)
	if err := expect("receivables aging", res, err); err != nil {
		return 0, err
	}
	totals, err := json.Marshal(res.JSON200.Totals)
	if err != nil {
		return 0, err
	}
	f.add("aging|%s|%d", totals, res.JSON200.Total)
	return res.JSON200.Total, nil
}

// alerts adds each variant under its minimum: location, SKU, stock, minimum,
// maximum and suggested quantity. It runs after stock, which names the
// locations.
func (f *fingerprint) alerts(ctx context.Context) (int, error) {
	for offset := 0; ; offset += pageSize {
		limit, off := pageSize, offset
		res, err := f.r.api.ListReorderAlertsWithResponse(ctx, &apiclient.ListReorderAlertsParams{Limit: &limit, Offset: &off}, f.owner)
		if err := expect("list reorder alerts", res, err); err != nil {
			return 0, err
		}
		for _, a := range res.JSON200.Items {
			f.add("alert|%s|%s|%s|%s|%s|%s", f.locs[a.LocationId], f.skus[a.VariantId],
				a.Quantity, a.MinQuantity, a.MaxQuantity, a.SuggestedQuantity)
		}
		if offset+pageSize >= res.JSON200.Total {
			return res.JSON200.Total, nil
		}
	}
}
