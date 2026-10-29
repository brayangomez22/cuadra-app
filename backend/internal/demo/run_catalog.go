package demo

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// ids are the ids the API gave to what the scenario names by index.
type ids struct {
	units      map[string]uuid.UUID
	categories map[string]uuid.UUID
	segments   map[string]uuid.UUID
	locations  []uuid.UUID
	variants   [][]uuid.UUID
	customers  []uuid.UUID
	suppliers  []uuid.UUID
}

// writeCatalog creates the locations and the products, after reading the
// units, categories and segments the business template seeded.
func (r *runner) writeCatalog(ctx context.Context) error {
	owner := r.as(r.s.Profile.OwnerEmail())
	if err := r.readTemplateData(ctx, owner); err != nil {
		return err
	}
	for _, name := range r.s.Locations {
		res, err := r.api.CreateLocationWithResponse(ctx, apiclient.LocationInput{Name: name}, owner)
		if err := expect("create location", res, err); err != nil {
			return err
		}
		r.ids.locations = append(r.ids.locations, res.JSON201.Id)
	}
	for i := range r.s.Products {
		vids, err := r.createProduct(ctx, owner, r.s.Products[i])
		if err != nil {
			return fmt.Errorf("product %q: %w", r.s.Products[i].Name, err)
		}
		r.ids.variants = append(r.ids.variants, vids)
	}
	return nil
}

// readTemplateData maps the template's units (by code), categories and
// customer segments (by name) to their ids.
func (r *runner) readTemplateData(ctx context.Context, owner apiclient.RequestEditorFn) error {
	units, err := r.api.ListUnitsWithResponse(ctx, owner)
	if err := expect("list units", units, err); err != nil {
		return err
	}
	cats, err := r.api.ListCategoriesWithResponse(ctx, owner)
	if err := expect("list categories", cats, err); err != nil {
		return err
	}
	segs, err := r.api.ListCustomerSegmentsWithResponse(ctx, owner)
	if err := expect("list segments", segs, err); err != nil {
		return err
	}
	r.ids.units, r.ids.categories, r.ids.segments = map[string]uuid.UUID{}, map[string]uuid.UUID{}, map[string]uuid.UUID{}
	for _, u := range units.JSON200.Items {
		r.ids.units[u.Code] = u.Id
	}
	for _, c := range cats.JSON200.Items {
		r.ids.categories[c.Name] = c.Id
	}
	for _, s := range segs.JSON200.Items {
		r.ids.segments[s.Name] = s.Id
	}
	return nil
}

// createProduct creates p, with its variants when it varies on axes, and
// returns its variants' ids in p's order.
func (r *runner) createProduct(ctx context.Context, owner apiclient.RequestEditorFn, p Product) ([]uuid.UUID, error) {
	attrs := apiclient.ProductAttributes{}
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	category := r.ids.categories[p.Category]
	tax := httpx.FormatDecimal(taxRate)
	if len(p.Variants[0].Axes) == 0 {
		v := p.Variants[0]
		res, err := r.api.CreateProductWithResponse(ctx, apiclient.ProductInput{
			Name: p.Name, Sku: v.SKU, Barcode: &v.Barcode, BaseUnitId: r.ids.units[p.Unit], CategoryId: &category,
			Cost: httpx.FormatMoney(v.Cost), Price: httpx.FormatMoney(v.Price), TaxRate: tax, Attributes: attrs,
		}, owner)
		if err := expect("create product", res, err); err != nil {
			return nil, err
		}
		return variantIDs(res.JSON201), nil
	}
	in := apiclient.ProductWithVariantsInput{
		Name: p.Name, BaseUnitId: r.ids.units[p.Unit], CategoryId: &category, TaxRate: tax, Attributes: attrs,
	}
	for _, v := range p.Variants {
		in.Variants = append(in.Variants, apiclient.ProductVariantInput{
			Sku: v.SKU, Barcode: &v.Barcode, AxisValues: v.Axes,
			Cost: httpx.FormatMoney(v.Cost), Price: httpx.FormatMoney(v.Price),
		})
	}
	res, err := r.api.CreateProductWithVariantsWithResponse(ctx, in, owner)
	if err := expect("create product with variants", res, err); err != nil {
		return nil, err
	}
	return variantIDs(res.JSON201), nil
}

func variantIDs(p *apiclient.Product) []uuid.UUID {
	out := make([]uuid.UUID, len(p.Variants))
	for i, v := range p.Variants {
		out[i] = v.Id
	}
	return out
}

// writePeople creates the customers (with their credit) and the suppliers.
func (r *runner) writePeople(ctx context.Context) error {
	owner := r.as(r.s.Profile.OwnerEmail())
	for _, c := range r.s.Customers {
		segment := r.ids.segments[c.Segment]
		res, err := r.api.CreateCustomerWithResponse(ctx, apiclient.CustomerInput{
			Name: c.Name, DocumentType: apiclient.DocumentType(c.DocumentType), DocumentNumber: c.Document,
			PersonType: apiclient.PersonType(c.PersonType), Phone: &c.Phone, City: &c.City, SegmentId: &segment,
		}, owner)
		if err := expect("create customer", res, err); err != nil {
			return fmt.Errorf("customer %q: %w", c.Name, err)
		}
		r.ids.customers = append(r.ids.customers, res.JSON201.Id)
		if !c.CreditLimit.IsPositive() {
			continue
		}
		credit, err := r.api.SetCustomerCreditWithResponse(ctx, res.JSON201.Id, apiclient.CreditTermsInput{
			CreditLimit: httpx.FormatMoney(c.CreditLimit), TermDays: c.TermDays,
		}, owner)
		if err := expect("set credit", credit, err); err != nil {
			return err
		}
	}
	for _, sup := range r.s.Suppliers {
		term := sup.TermDays
		res, err := r.api.CreateSupplierWithResponse(ctx, apiclient.SupplierInput{
			Name: sup.Name, Nit: withCheckDigit(sup.NIT), PaymentTermDays: &term,
		}, owner)
		if err := expect("create supplier", res, err); err != nil {
			return err
		}
		r.ids.suppliers = append(r.ids.suppliers, res.JSON201.Id)
	}
	return nil
}
