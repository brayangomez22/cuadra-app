package httpadapter

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http/importsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func importBody(imp *domain.Import) importsapi.Import {
	c := imp.Counts()
	return importsapi.Import{
		Id:            imp.ID(),
		FileName:      imp.FileName(),
		Format:        importsapi.ImportFormat(imp.Format()),
		Status:        importsapi.ImportStatus(imp.Status()),
		OnExistingSku: importsapi.ExistingSkuPolicy(imp.Policy()),
		LocationId:    imp.LocationID(),
		Counts: importsapi.ImportCounts{
			Rows: c.Rows, Products: c.Products, Created: c.Created,
			Updated: c.Updated, Skipped: c.Skipped, Errors: c.Errors,
		},
		CreatedAt:  imp.CreatedAt(),
		StartedAt:  imp.StartedAt(),
		FinishedAt: imp.FinishedAt(),
	}
}

func previewBody(p app.Preview) importsapi.ImportPreview {
	sample := make([]importsapi.ImportProductPreview, 0, len(p.Sample))
	for _, g := range p.Sample {
		sample = append(sample, productPreview(g))
	}
	return importsapi.ImportPreview{Import: importBody(p.Import), Errors: rowErrorsBody(p.Errors), Sample: sample}
}

func productPreview(g domain.Group) importsapi.ImportProductPreview {
	variants := make([]importsapi.ImportVariantPreview, 0, len(g.Variants))
	for _, v := range g.Variants {
		body := importsapi.ImportVariantPreview{
			Row: v.Row, Sku: v.SKU, Price: httpx.FormatDecimal(v.Price), AxisValues: map[string]string{},
		}
		for axis, value := range v.Axes {
			body.AxisValues[axis] = value
		}
		if v.HasStock {
			stock := httpx.FormatDecimal(v.Stock)
			body.InitialStock = &stock
		}
		variants = append(variants, body)
	}
	return importsapi.ImportProductPreview{
		Row: g.Row, Name: g.Product.Name, Category: g.Product.Category, Unit: g.Product.Unit, Variants: variants,
	}
}

func rowErrorsBody(errs []domain.RowError) []importsapi.ImportRowError {
	out := make([]importsapi.ImportRowError, 0, len(errs))
	for _, e := range errs {
		out = append(out, importsapi.ImportRowError{
			Row: e.Row, Column: e.Column, Code: e.Code, Message: rowMessage(e),
		})
	}
	return out
}
