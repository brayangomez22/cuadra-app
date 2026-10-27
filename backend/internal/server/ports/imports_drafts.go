package ports

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	catalogdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	importsapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	importsdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// drafts translates the groups of a file into the catalog's drafts.
func drafts(groups []importsdomain.Group) []catalogapp.ProductDraft {
	out := make([]catalogapp.ProductDraft, 0, len(groups))
	for _, g := range groups {
		d := catalogapp.ProductDraft{
			Name: g.Product.Name, Description: g.Product.Description, Category: g.Product.Category,
			UnitCode: g.Product.Unit, TaxRate: g.Product.TaxRate,
			Attributes: make(map[string]catalogdomain.AttributeValue, len(g.Product.Attributes)),
		}
		for key, v := range g.Product.Attributes {
			if v.IsBool {
				d.Attributes[key] = catalogdomain.BoolValue(v.Bool)
			} else {
				d.Attributes[key] = catalogdomain.TextValue(v.Text)
			}
		}
		for _, v := range g.Variants {
			d.Variants = append(d.Variants, catalogapp.VariantInput{
				SKU: v.SKU, Barcode: v.Barcode, Cost: v.Cost, Price: v.Price, AxisValues: v.Axes,
			})
		}
		out = append(out, d)
	}
	return out
}

// productResults translates the catalog's results. A draft's error the
// catalog describes is the group's problem; any other is an infrastructure
// failure and is returned.
func productResults(results []catalogapp.DraftResult) ([]importsapp.ProductResult, error) {
	out := make([]importsapp.ProductResult, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			code, message, ok := catalog.DescribeError(r.Err)
			if !ok {
				return nil, r.Err
			}
			out = append(out, importsapp.ProductResult{Problem: &importsapp.Problem{Code: code, Message: message}})
			continue
		}
		pr := importsapp.ProductResult{Outcome: importsapp.Outcome(r.Outcome)}
		for _, v := range r.Variants {
			pr.Variants = append(pr.Variants, importsapp.VariantResult{ID: v.ID, Created: v.Created})
		}
		out = append(out, pr)
	}
	return out, nil
}
