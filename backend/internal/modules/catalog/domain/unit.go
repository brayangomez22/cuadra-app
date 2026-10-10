package domain

// UnitOfMeasure is the base unit a product is stocked and sold in. Quantities
// are decimal, so fractional units (2.5 m) need no special unit.
type UnitOfMeasure string

// The initial fixed catalog of units. Codes are lowercase and without
// accents; the UI shows the label ("galón").
const (
	UnitPiece  UnitOfMeasure = "und"
	UnitMeter  UnitOfMeasure = "m"
	UnitKilo   UnitOfMeasure = "kg"
	UnitLiter  UnitOfMeasure = "l"
	UnitBox    UnitOfMeasure = "caja"
	UnitRoll   UnitOfMeasure = "rollo"
	UnitBulto  UnitOfMeasure = "bulto"
	UnitGallon UnitOfMeasure = "galon"
)

var units = []UnitOfMeasure{UnitPiece, UnitMeter, UnitKilo, UnitLiter, UnitBox, UnitRoll, UnitBulto, UnitGallon}

// ParseUnit returns the unit with code s. Matching is exact (lowercase).
func ParseUnit(s string) (UnitOfMeasure, error) {
	u := UnitOfMeasure(s)
	if !u.valid() {
		return "", ErrInvalidUnit
	}
	return u, nil
}

// Units returns the catalog of units, in display order.
func Units() []UnitOfMeasure {
	out := make([]UnitOfMeasure, len(units))
	copy(out, units)
	return out
}

func (u UnitOfMeasure) valid() bool {
	for _, known := range units {
		if u == known {
			return true
		}
	}
	return false
}

func (u UnitOfMeasure) String() string { return string(u) }
