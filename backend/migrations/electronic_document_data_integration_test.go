//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// invoicingAdded is 00031_invoicing; electronicDocumentDataAdded is
// 00032_electronic_document_data: the DANE municipalities, the units' DIAN
// code, the IVA treatment and the codes frozen in each sale line, and the
// customers' new document types and municipality.
const (
	invoicingAdded              = 31
	electronicDocumentDataAdded = 32
)

// documentSeed is a tenant before 00032 with two units, a taxed product
// sold by meter, an untaxed one sold by box, and a sale of both.
type documentSeed struct {
	tenant, taxed, untaxed, sale uuid.UUID
}

func seedDocumentData(t *testing.T, s *dbtest.Server) documentSeed {
	t.Helper()
	conn := connect(t, s.SuperuserURL)
	d := documentSeed{tenant: seedTenantWith(t, s, "ferreteria")}
	id := func() uuid.UUID { return uuid.Must(uuid.NewV7()) }
	meter, box, location, user := id(), id(), id(), id()
	d.taxed, d.untaxed, d.sale = id(), id(), id()
	taxedVariant, untaxedVariant := id(), id()
	exec := func(sql string, args ...any) {
		t.Helper()
		_, err := conn.Exec(t.Context(), sql, args...)
		require.NoError(t, err, sql)
	}
	exec(`INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
	      VALUES ($1, $3, 'm', 'Metro', now()), ($2, $3, 'caja', 'Caja', now())`, meter, box, d.tenant)
	exec(`INSERT INTO products (id, tenant_id, name, description, tax_rate, active, created_at, base_unit_id)
	      VALUES ($1, $3, 'Cable #12', '', 0.19, true, now(), $4), ($2, $3, 'Cuaderno', '', 0, true, now(), $5)`,
		d.taxed, d.untaxed, d.tenant, meter, box)
	exec(`INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, is_default, active, created_at)
	      VALUES ($1, $3, $4, 'CAB-12', 1000, 2000, true, true, now()), ($2, $3, $5, 'CUA-100', 2000, 4000, true, true, now())`,
		taxedVariant, untaxedVariant, d.tenant, d.taxed, d.untaxed)
	exec(`INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, 'Principal', true, now())`,
		location, d.tenant)
	exec(`INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at)
	      VALUES ($1, $2, 'duena@negocio.co', 'Dueña', 'x', 'owner', true, now())`, user, d.tenant)
	exec(`INSERT INTO sales (id, tenant_id, location_id, number, user_id, status, subtotal, discounts, tax, total, paid,
	                         change, created_at, confirmed_at)
	      VALUES ($1, $2, $3, 1, $4, 'confirmed', 8000, 0, 760, 8760, 8760, 0, now(), now())`,
		d.sale, d.tenant, location, user)
	exec(`INSERT INTO sale_lines (tenant_id, sale_id, position, product_id, product_name, quantity, unit_price, tax_rate,
	                              discount, gross, tax, variant_id, price_rule)
	      VALUES ($1, $2, 0, $3, 'Cable #12', 2, 2000, 0.19, 0, 4000, 760, $4, 'variant_price'),
	             ($1, $2, 1, $5, 'Cuaderno', 1, 4000, 0, 0, 4000, 0, $6, 'variant_price')`,
		d.tenant, d.sale, d.taxed, taxedVariant, d.untaxed, untaxedVariant)
	return d
}

func TestElectronicDocumentDataMigration(t *testing.T) {
	t.Run("siembra los 1.122 municipios del DANE y la app solo los lee", func(t *testing.T) {
		s := dbtest.StartAt(t, invoicingAdded)
		s.MigrateTo(t, electronicDocumentDataAdded)

		app := connect(t, s.AppURL)
		var name, department string
		var count int
		require.NoError(t, app.QueryRow(t.Context(), "SELECT count(*) FROM municipalities").Scan(&count))
		require.Equal(t, 1122, count)
		require.NoError(t, app.QueryRow(t.Context(),
			"SELECT name, department_name FROM municipalities WHERE code = '05001'").Scan(&name, &department))
		require.Equal(t, "Medellín", name)
		require.Equal(t, "Antioquia", department)
		_, err := app.Exec(t.Context(), "UPDATE municipalities SET name = 'Otro' WHERE code = '05001'")
		require.Error(t, err, "la app no cambia el catálogo")
	})

	t.Run("da el código DIAN a las unidades conocidas y el tratamiento y los códigos a lo vendido", func(t *testing.T) {
		s := dbtest.StartAt(t, invoicingAdded)
		d := seedDocumentData(t, s)

		s.MigrateTo(t, electronicDocumentDataAdded)

		conn := connect(t, s.SuperuserURL)
		var meter, box *string
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT (SELECT dian_code FROM units_of_measure WHERE tenant_id = $1 AND code = 'm'),
			       (SELECT dian_code FROM units_of_measure WHERE tenant_id = $1 AND code = 'caja')`, d.tenant).
			Scan(&meter, &box))
		require.Equal(t, "MTR", *meter)
		require.Nil(t, box, "una unidad sin código conocido va como 94 en la factura")

		var taxed, untaxed string
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT (SELECT tax_treatment FROM products WHERE id = $1), (SELECT tax_treatment FROM products WHERE id = $2)`,
			d.taxed, d.untaxed).Scan(&taxed, &untaxed))
		require.Equal(t, "taxed", taxed)
		require.Equal(t, "excluded", untaxed)

		rows, err := conn.Query(t.Context(), `
			SELECT sku, base_unit_code, unit_dian_code, tax_treatment FROM sale_lines WHERE sale_id = $1 ORDER BY position`,
			d.sale)
		require.NoError(t, err)
		var got [][4]string
		for rows.Next() {
			var sku, base, treatment string
			var dian *string
			require.NoError(t, rows.Scan(&sku, &base, &dian, &treatment))
			require.Nil(t, dian)
			got = append(got, [4]string{sku, base, "", treatment})
		}
		require.NoError(t, rows.Err())
		require.Equal(t, [][4]string{{"CAB-12", "m", "", "taxed"}, {"CUA-100", "caja", "", "excluded"}}, got)
	})

	t.Run("la migración deja FORCE en las tablas que toca", func(t *testing.T) {
		s := dbtest.StartAt(t, invoicingAdded)
		seedDocumentData(t, s)
		s.MigrateTo(t, electronicDocumentDataAdded)

		conn := connect(t, s.SuperuserURL)
		for _, table := range []string{"units_of_measure", "products", "product_variants", "sale_lines", "customers"} {
			var forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&forced))
			require.True(t, forced, table)
		}
	})

	t.Run("un producto gravado necesita tasa y uno exento o excluido tasa 0", func(t *testing.T) {
		s := dbtest.StartAt(t, invoicingAdded)
		d := seedDocumentData(t, s)
		s.MigrateTo(t, electronicDocumentDataAdded)

		conn := connect(t, s.SuperuserURL)
		_, err := conn.Exec(t.Context(), "UPDATE products SET tax_treatment = 'exempt' WHERE id = $1", d.taxed)
		require.Error(t, err)
		_, err = conn.Exec(t.Context(), "UPDATE products SET tax_treatment = 'exempt' WHERE id = $1", d.untaxed)
		require.NoError(t, err)
	})

	t.Run("el Down quita lo agregado y falla con clientes de los tipos nuevos", func(t *testing.T) {
		s := dbtest.StartAt(t, invoicingAdded)
		d := seedDocumentData(t, s)
		s.MigrateTo(t, electronicDocumentDataAdded)

		s.MigrateDownTo(t, invoicingAdded)

		conn := connect(t, s.SuperuserURL)
		var table, column bool
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT to_regclass('public.municipalities') IS NOT NULL,
			       EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sale_lines' AND column_name = 'sku')`).
			Scan(&table, &column))
		require.False(t, table)
		require.False(t, column)

		s.MigrateTo(t, electronicDocumentDataAdded)
		_, err := conn.Exec(t.Context(), `
			INSERT INTO customers (id, tenant_id, person_type, document_type, document_number, name, tax_responsibilities,
			                       active, created_at, updated_at, municipality_code)
			VALUES ($1, $2, 'legal', 'foreign_nit', 'B12345678', 'Empresa', '{R-99-PN}', true, now(), now(), '11001')`,
			uuid.Must(uuid.NewV7()), d.tenant)
		require.NoError(t, err)
		require.Error(t, s.TryMigrateDownTo(t, invoicingAdded))
	})
}
