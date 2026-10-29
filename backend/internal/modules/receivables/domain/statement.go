package domain

import (
	"cmp"
	"slices"

	"github.com/shopspring/decimal"
)

// kindOrder orders movements at the same instant: what is owed before what
// takes it off.
var kindOrder = map[MovementKind]int{MovementCharge: 0, MovementPayment: 1, MovementCancellation: 2}

// Statement returns a customer's movements newest first, each with the
// balance it left: a charge adds its amount, a payment or a cancellation
// takes it off.
func Statement(movements []Movement) []Movement {
	out := slices.Clone(movements)
	slices.SortStableFunc(out, func(x, y Movement) int {
		if c := x.At.Compare(y.At); c != 0 {
			return c
		}
		return cmp.Compare(kindOrder[x.Kind], kindOrder[y.Kind])
	})
	balance := decimal.Zero
	for i := range out {
		if out[i].Kind == MovementCharge {
			balance = balance.Add(out[i].Amount)
		} else {
			balance = balance.Sub(out[i].Amount)
		}
		out[i].Balance = balance
	}
	slices.Reverse(out)
	return out
}
