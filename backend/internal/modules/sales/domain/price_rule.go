package domain

const maxPriceRuleLength = 64

// PriceRuleID is the stable id of the price rule that set a line's price:
// "variant_price" for the variant's own price, or the id a module gives its
// rule ("price_lists", "volume_pricing"...). It is lowercase snake_case, so
// it reads the same in a report years later; it is text in the database,
// not an enum, because modules add rules.
type PriceRuleID string

// VariantPriceRule is the base rule: the variant's price, which applies when
// no other rule does.
const VariantPriceRule PriceRuleID = "variant_price"

// NewPriceRuleID validates s: a lowercase letter followed by lowercase
// letters, digits or underscores, at most 64 characters.
func NewPriceRuleID(s string) (PriceRuleID, error) {
	if s == "" || len(s) > maxPriceRuleLength || s[0] < 'a' || s[0] > 'z' {
		return "", ErrInvalidPriceRule
	}
	for _, c := range []byte(s) {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return "", ErrInvalidPriceRule
		}
	}
	return PriceRuleID(s), nil
}
