package app_test

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
)

// fakeBalances is the receivables port: what each customer owes, and the
// customers it was last asked about.
type fakeBalances struct {
	owed  map[uuid.UUID]app.Balance
	asked []uuid.UUID
}

func (f *fakeBalances) Balances(_ context.Context, _, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.Balance, error) {
	f.asked = ids
	out := map[uuid.UUID]app.Balance{}
	for _, id := range ids {
		if b, ok := f.owed[id]; ok {
			out[id] = b
		}
	}
	return out, nil
}
