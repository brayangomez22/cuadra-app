//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// customersAdded is 00024_customers: the customers, their segments and the
// sales' foreign key to them.
const customersAdded = 24

// ferreteriaSegments are the customer segments of the ferreteria template,
// the one the existing tenants have (00018).
var ferreteriaSegments = []string{"Contratista", "Empresa", "Maestro de obra", "Público"}

func TestCustomersMigration(t *testing.T) {
	t.Run("los tenants existentes reciben los segmentos de la plantilla ferreteria", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		tenants := seedTenants(t, s, 2)

		s.MigrateTo(t, customersAdded)

		conn := connect(t, s.SuperuserURL)
		for _, tenant := range tenants {
			rows, err := conn.Query(t.Context(),
				"SELECT name FROM customer_segments WHERE tenant_id = $1 ORDER BY name", tenant)
			require.NoError(t, err)
			var names []string
			for rows.Next() {
				var name string
				require.NoError(t, rows.Scan(&name))
				names = append(names, name)
			}
			require.NoError(t, rows.Err())
			require.Equal(t, ferreteriaSegments, names)
		}
		var v7 int
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM customer_segments WHERE substring(id::text, 15, 1) = '7'").Scan(&v7))
		require.Equal(t, 2*len(ferreteriaSegments), v7, "the ids are UUID v7")
	})

	t.Run("la migración deja FORCE en las tablas que toca", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		seedTenants(t, s, 1)

		s.MigrateTo(t, customersAdded)

		for _, table := range []string{"tenants", "customer_segments", "customers"} {
			var enabled, forced bool
			require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
				"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&enabled, &forced))
			require.True(t, enabled, table)
			require.True(t, forced, table)
		}
	})

	t.Run("una persona jurídica solo se guarda con NIT", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		tenant := seedTenants(t, s, 1)[0]
		s.MigrateTo(t, customersAdded)
		conn := connect(t, s.SuperuserURL)

		insert := `INSERT INTO customers (id, tenant_id, person_type, document_type, document_number, name,
		                                  tax_responsibilities, active, created_at, updated_at)
		           VALUES ($1, $2, 'legal', $3, $4, 'Empresa', '{R-99-PN}', true, now(), now())`
		_, err := conn.Exec(t.Context(), insert, uuid.Must(uuid.NewV7()), tenant, "cc", "1020304050")
		require.Error(t, err)
		_, err = conn.Exec(t.Context(), insert, uuid.Must(uuid.NewV7()), tenant, "nit", "900123456-8")
		require.NoError(t, err)
	})

	t.Run("el Down borra los clientes, los segmentos y la referencia de las ventas", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		seedTenants(t, s, 1)
		s.MigrateTo(t, customersAdded)

		s.MigrateDownTo(t, customersAdded-1)

		var tables, fks int
		conn := connect(t, s.SuperuserURL)
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM pg_class WHERE relname IN ('customers', 'customer_segments')").Scan(&tables))
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM pg_constraint WHERE conname = 'sales_customer_fkey'").Scan(&fks))
		require.Zero(t, tables)
		require.Zero(t, fks)
	})
}
