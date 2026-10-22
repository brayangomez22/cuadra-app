package modules

// IDs of Cuadra's modules.
const (
	// Variants lets a product have several variants (size, color,
	// presentation...). Without it, each product has only its default
	// variant, which can still be edited.
	Variants = "variants"
	// UnitConversions lets a tenant give a variant alternate units (a box
	// of 12, a roll of 100 m) with their factor and price, to sell and buy
	// in them. Without it, the units already defined still sell.
	UnitConversions = "unit_conversions"
)

// Builtin returns the definitions of Cuadra's modules, in the order they are
// presented. A new module is added here.
func Builtin() []Definition {
	return []Definition{
		{ID: Variants, Name: "Variantes"},
		{ID: UnitConversions, Name: "Unidades alternativas"},
	}
}
