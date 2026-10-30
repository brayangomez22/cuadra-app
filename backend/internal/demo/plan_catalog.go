package demo

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/shopspring/decimal"
)

// brandAttribute is the attribute key of the brand, which every demo
// template defines.
const brandAttribute = "marca"

// combo is one article of a family: a name, a spec and a brand.
type combo struct{ name, spec, brand int }

// catalogPlan builds the articles with unique SKUs and barcodes.
type catalogPlan struct {
	rng      *rand.Rand
	profile  Profile
	variants int
}

// planCatalog returns products whose variants add up to target, taking the
// families in turns so every category is in the catalog.
func planCatalog(rng *rand.Rand, p Profile, target int) ([]Product, error) {
	c := &catalogPlan{rng: rng, profile: p}
	combos := make([][]combo, len(p.Families))
	for i, f := range p.Families {
		combos[i] = c.combos(f)
	}
	var products []Product
	for turn := 0; c.variants < target; turn++ {
		added := false
		for fi, f := range p.Families {
			if turn >= len(combos[fi]) || c.variants >= target {
				continue
			}
			products = append(products, c.product(fi, f, combos[fi][turn], target-c.variants))
			added = true
		}
		if !added {
			return nil, fmt.Errorf("demo: profile %s has articles for %d variants, not %d", p.Template, c.variants, target)
		}
	}
	return products, nil
}

// combos returns every article of f, shuffled.
func (c *catalogPlan) combos(f Family) []combo {
	out := make([]combo, 0, len(f.Names)*len(f.Specs)*len(f.Brands))
	for n := range f.Names {
		for s := range f.Specs {
			for b := range f.Brands {
				out = append(out, combo{n, s, b})
			}
		}
	}
	c.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// product builds the article of combo k with at most room variants.
func (c *catalogPlan) product(fi int, f Family, k combo, room int) Product {
	attrs := map[string]string{brandAttribute: f.Brands[k.brand]}
	for key, v := range f.Attributes {
		attrs[key] = v
	}
	step := decimal.NewFromInt(1)
	if f.Step != "" {
		step = decimal.RequireFromString(f.Step)
	}
	price := c.price(f)
	cost := price.Mul(decimal.NewFromInt(int64(55 + c.rng.IntN(18)))).Div(decimal.NewFromInt(100)).Round(-1)
	axes := c.axes(f)
	if len(axes) > room {
		axes = axes[:room]
	}
	if len(axes) == 0 {
		axes = []map[string]string{{}}
	}
	variants := make([]Variant, len(axes))
	for i, a := range axes {
		variants[i] = c.variant(f.Category, a, cost, price)
	}
	product := Product{
		Name:     strings.Join([]string{f.Names[k.name], f.Specs[k.spec], f.Brands[k.brand]}, " "),
		Category: f.Category, Unit: f.Unit, Attributes: attrs, Variants: variants,
		Supplier: fi % len(c.profile.Suppliers), Step: step, MaxQuantity: f.Quantity,
	}
	if u := f.PurchaseUnit; u != nil {
		product.PurchaseUnit, product.PurchaseFactor = u.Unit, decimal.NewFromInt(int64(u.Factor))
	}
	return product
}

// price is a price of f's range, in hundreds of pesos.
func (c *catalogPlan) price(f Family) decimal.Decimal {
	hundreds := f.Price[0]/100 + c.rng.IntN(f.Price[1]/100-f.Price[0]/100+1)
	return decimal.NewFromInt(int64(hundreds * 100))
}

// axes chooses the axis values of an article: a run of the first axis
// (sizes) and one or two values of each other (colors). A single axis is a
// choice of one to three values.
func (c *catalogPlan) axes(f Family) []map[string]string {
	if len(f.Axes) == 0 {
		return nil
	}
	first := f.Axes[0].Values
	var chosen [][]string
	if len(f.Axes) == 1 {
		chosen = append(chosen, c.pick(first, 1+c.rng.IntN(3)))
	} else {
		n := min(len(first), 3+c.rng.IntN(2))
		start := c.rng.IntN(len(first) - n + 1)
		chosen = append(chosen, first[start:start+n])
	}
	for _, a := range f.Axes[1:] {
		chosen = append(chosen, c.pick(a.Values, 1+c.rng.IntN(2)))
	}
	out := []map[string]string{{}}
	for i, values := range chosen {
		var next []map[string]string
		for _, prev := range out {
			for _, v := range values {
				m := map[string]string{f.Axes[i].Name: v}
				for k, pv := range prev {
					m[k] = pv
				}
				next = append(next, m)
			}
		}
		out = next
	}
	return out
}

// pick returns n distinct values in their order.
func (c *catalogPlan) pick(values []string, n int) []string {
	idx := c.rng.Perm(len(values))[:min(n, len(values))]
	out := make([]string, 0, len(idx))
	for i, v := range values {
		for _, j := range idx {
			if i == j {
				out = append(out, v)
			}
		}
	}
	return out
}

// variant numbers a new variant: SKU by category and an EAN-13 with the
// Colombian prefix.
func (c *catalogPlan) variant(category string, axes map[string]string, cost, price decimal.Decimal) Variant {
	c.variants++
	return Variant{
		SKU:     fmt.Sprintf("%s-%05d", skuPrefix(category), c.variants),
		Barcode: ean13(fmt.Sprintf("7700%08d", c.variants)),
		Axes:    axes, Cost: cost, Price: price,
	}
}

// skuPrefix is the first three letters of category, in uppercase ASCII.
func skuPrefix(category string) string {
	fold := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n")
	var b strings.Builder
	for _, r := range fold.Replace(strings.ToLower(category)) {
		if r >= 'a' && r <= 'z' && b.Len() < 3 {
			b.WriteRune(r - 'a' + 'A')
		}
	}
	return b.String()
}

// ean13 appends the check digit to 12 digits.
func ean13(digits string) string {
	sum := 0
	for i, r := range digits {
		w := 1
		if i%2 == 1 {
			w = 3
		}
		sum += int(r-'0') * w
	}
	return fmt.Sprintf("%s%d", digits, (10-sum%10)%10)
}
