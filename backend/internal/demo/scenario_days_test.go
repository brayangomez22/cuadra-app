package demo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// minuteOf is the minute of the day of t, in its own zone.
func minuteOf(t time.Time) int { return t.Hour()*60 + t.Minute() }

func TestScenarioDays(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("la caja abre hacia las 8 y cierra hacia las 7:30, no siempre al mismo minuto: "+template, func(t *testing.T) {
			s := small(t, template, 2)
			openings, closings := map[int]bool{}, map[int]bool{}
			for _, d := range s.Days {
				open, closing := minuteOf(d.Opening), minuteOf(d.Closing)
				require.GreaterOrEqual(t, open, 7*60+45)
				require.LessOrEqual(t, open, 8*60+15)
				require.GreaterOrEqual(t, closing, 19*60+15)
				require.LessOrEqual(t, closing, 19*60+50)
				openings[open], closings[closing] = true, true
				for _, e := range d.Events() {
					if e.Purchase == nil {
						require.True(t, e.At.After(d.Opening) && e.At.Before(d.Closing), "%s outside %s–%s", e.At, d.Opening, d.Closing)
					}
				}
			}
			require.Greater(t, len(openings), 5)
			require.Greater(t, len(closings), 5)
		})

		t.Run("la mayoría de las cajas cuadra y algunas cierran con un faltante o sobrante en miles: "+template, func(t *testing.T) {
			s := small(t, template, 2)
			square, off := 0, 0
			for _, d := range s.Days {
				require.Len(t, d.CashDifference, len(s.Locations))
				for _, diff := range d.CashDifference {
					if diff.IsZero() {
						square++
						continue
					}
					off++
					require.True(t, diff.Mod(decimalThousand).IsZero(), "a difference of %s", diff)
				}
			}
			require.Positive(t, off)
			require.Greater(t, square, off*4)
		})
	}
}
