package demo

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

var (
	firstNames = []string{
		"Ana", "Luis", "María", "Carlos", "Diana", "Jorge", "Paola", "Andrés", "Sandra", "Felipe",
		"Claudia", "Juan", "Natalia", "Óscar", "Lucía", "Camilo", "Marcela", "Hernán", "Valentina", "Javier",
	}
	lastNames = []string{
		"Gómez", "Rodríguez", "Martínez", "López", "García", "Hernández", "Pérez", "Ramírez", "Torres", "Díaz",
		"Moreno", "Vargas", "Rojas", "Castro", "Ortiz", "Sánchez", "Restrepo", "Cárdenas", "Ospina", "Muñoz",
	}
	companyWords = []string{"Inversiones", "Construcciones", "Comercializadora", "Distribuciones", "Servicios", "Soluciones"}
	companyNames = []string{"del Valle", "Andina", "La Esperanza", "El Progreso", "Santa Rita", "Los Pinos", "Altamira", "San José"}
	cities       = []string{"Bogotá", "Medellín", "Cali", "Barranquilla", "Bucaramanga", "Pereira", "Ibagué", "Manizales"}
)

// planCustomers returns n customers with unique documents: one in seven a
// company with a NIT, the CreditShare first with credit in the
// CreditSegment, and one in DefaulterOne of those never paying.
func planCustomers(rng *rand.Rand, p Profile, n int) []Customer {
	withCredit := max(1, n*p.CreditShare/100)
	out := make([]Customer, n)
	for i := range out {
		c := person(rng, i)
		if i%7 == 6 {
			c = company(rng, i)
		}
		c.Phone = fmt.Sprintf("3%02d%07d", 10+rng.IntN(40), rng.IntN(10_000_000))
		c.City = cities[rng.IntN(len(cities))]
		c.Segment = p.Segments[rng.IntN(len(p.Segments))]
		if i < withCredit {
			c.Segment = p.CreditSegment
			c.CreditLimit = decimal.NewFromInt(int64(5+rng.IntN(26)) * 100_000)
			c.TermDays = 15 * (1 + rng.IntN(2))
			c.Defaulter = p.DefaulterOne > 0 && i%p.DefaulterOne == p.DefaulterOne-1
		}
		out[i] = c
	}
	return out
}

// person is a natural person with a cédula; i keeps it unique.
func person(rng *rand.Rand, i int) Customer {
	name := fmt.Sprintf("%s %s %s", firstNames[rng.IntN(len(firstNames))],
		lastNames[rng.IntN(len(lastNames))], lastNames[rng.IntN(len(lastNames))])
	return Customer{
		Name: name, DocumentType: "cc", PersonType: "natural",
		Document: strconv.Itoa(1_010_000_000 + i*7919 + rng.IntN(7000)),
	}
}

// company is a legal person with a NIT and its check digit.
func company(rng *rand.Rand, i int) Customer {
	name := fmt.Sprintf("%s %s S.A.S.", companyWords[rng.IntN(len(companyWords))], companyNames[rng.IntN(len(companyNames))])
	return Customer{
		Name: name, DocumentType: "nit", PersonType: "legal",
		Document: withCheckDigit(strconv.Itoa(800_000_000 + i*104_729)),
	}
}

// withCheckDigit returns number-DV, the DV that taxid accepts.
func withCheckDigit(number string) string {
	for dv := range 10 {
		s := fmt.Sprintf("%s-%d", number, dv)
		if _, err := taxid.ParseNIT(s); err == nil {
			return s
		}
	}
	// Remainders 0 and 1 give a DV of 0 or 1, so one of the ten matches.
	panic("demo: no check digit for " + number)
}
