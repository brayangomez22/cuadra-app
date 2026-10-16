package modules

// IDs of Cuadra's modules.
const (
	// Variants lets a product have several variants (size, color,
	// presentation...). Without it, each product has only its default
	// variant, which can still be edited.
	Variants = "variants"
)

// Builtin returns the definitions of Cuadra's modules, in the order they are
// presented. A new module is added here.
func Builtin() []Definition {
	return []Definition{
		{ID: Variants, Name: "Variantes"},
	}
}
