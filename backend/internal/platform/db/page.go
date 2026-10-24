package db

import (
	"errors"
	"fmt"
	"math"
)

// ErrPageOutOfRange means a page does not fit the queries' int32 parameters.
var ErrPageOutOfRange = errors.New("db: page out of range")

// PageBounds converts a page to the int32 limit and offset of a query.
func PageBounds(limit, offset int) (int32, int32, error) {
	if limit < 0 || limit > math.MaxInt32 || offset < 0 || offset > math.MaxInt32 {
		return 0, 0, fmt.Errorf("%w (limit %d, offset %d)", ErrPageOutOfRange, limit, offset)
	}
	return int32(limit), int32(offset), nil
}
