package domain

import (
	"strings"

	"github.com/shopspring/decimal"
)

// NumberStyle is how a file writes its decimal numbers.
type NumberStyle int

// The styles: PlainNumbers uses a decimal point and optional comma
// thousands (12,500.50, what a spreadsheet stores); DecimalComma uses a
// decimal comma and optional point thousands (12.500,50, how a CSV saved by
// a spreadsheet in Spanish writes them).
const (
	PlainNumbers NumberStyle = iota
	DecimalComma
)

// ParseNumber reads a decimal number in style, ignoring spaces and a "$"
// sign; ok is false when s is not one.
func ParseNumber(s string, style NumberStyle) (_ decimal.Decimal, ok bool) {
	s = strings.NewReplacer(" ", "", " ", "", "$", "").Replace(s)
	thousands, point := ",", "."
	if style == DecimalComma {
		thousands, point = ".", ","
	}
	s = strings.ReplaceAll(s, thousands, "")
	if strings.Count(s, point) > 1 || strings.ContainsAny(s, "eE") {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(strings.Replace(s, point, ".", 1))
	if err != nil {
		return decimal.Zero, false
	}
	return d, true
}

// ParseTaxRate reads an IVA as a percentage ("19", "19 %") or as a rate
// ("0.19"): a number above 1 is a percentage. ok is false when s is not a
// number.
func ParseTaxRate(s string, style NumberStyle) (_ decimal.Decimal, ok bool) {
	percent := strings.Contains(s, "%")
	d, ok := ParseNumber(strings.ReplaceAll(s, "%", ""), style)
	if !ok {
		return decimal.Zero, false
	}
	if percent || d.GreaterThan(decimal.NewFromInt(1)) {
		d = d.Shift(-2)
	}
	return d, true
}
