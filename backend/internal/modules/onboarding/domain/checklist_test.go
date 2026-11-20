package domain_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

// The getting-started steps of two kinds of business, as their templates
// declare them: a hardware store with alternate units and credit customers,
// and a clothing store with sizes and colors.
var (
	hardwareSteps = []string{"business_profile", "invoicing", "products", "alternate_units", "initial_stock", "customer_credit", "invite_team", "first_sale"}
	clothingSteps = []string{"business_profile", "invoicing", "products", "variant_products", "initial_stock", "invite_team", "first_sale"}
)

// owner can resolve every step.
func owner(string) bool { return true }

// only grants exactly the permissions given.
func only(perms ...string) func(string) bool {
	return func(p string) bool { return slices.Contains(perms, p) }
}

func stepIDs(c domain.Checklist) []domain.StepID {
	ids := make([]domain.StepID, 0, len(c.Steps))
	for _, s := range c.Steps {
		ids = append(ids, s.ID)
	}
	return ids
}

func stepDone(t *testing.T, c domain.Checklist, id domain.StepID) bool {
	t.Helper()
	for _, s := range c.Steps {
		if s.ID == id {
			return s.Done
		}
	}
	require.Failf(t, "falta el paso", "%s no está en la lista", id)
	return false
}

// allDone are the facts of a tenant that has done everything.
var allDone = domain.Facts{
	HasContact: true, InvoicingEnabled: true, InvoicingReady: true, HasProduct: true, HasStock: true,
	HasAlternateUnit: true, HasVariantProduct: true, HasReorderRule: true, HasCreditCustomer: true,
	HasTeammate: true, HasSale: true,
}

func TestBuildChecklist_StepDoneOnlyWithItsData(t *testing.T) {
	cases := []struct {
		name  string
		steps []string
		facts domain.Facts
		step  domain.StepID
		done  bool
	}{
		{"ferretería: sin ventas la primera venta queda pendiente", hardwareSteps, domain.Facts{}, domain.StepFirstSale, false},
		{"ferretería: con una venta confirmada la primera venta está completa", hardwareSteps, domain.Facts{HasSale: true}, domain.StepFirstSale, true},
		{"ropa: sin productos el paso de productos queda pendiente", clothingSteps, domain.Facts{}, domain.StepProducts, false},
		{"ropa: con un producto el paso de productos está completo", clothingSteps, domain.Facts{HasProduct: true}, domain.StepProducts, true},
		{"ropa: un producto sin variantes no completa el de variantes", clothingSteps, domain.Facts{HasProduct: true}, domain.StepVariantProducts, false},
		{"ropa: un producto con variantes completa el de variantes", clothingSteps, domain.Facts{HasVariantProduct: true}, domain.StepVariantProducts, true},
		{"ferretería: datos del negocio incompletos quedan pendientes", hardwareSteps, domain.Facts{}, domain.StepBusinessProfile, false},
		{"ferretería: con dirección, ciudad y teléfono los datos están completos", hardwareSteps, domain.Facts{HasContact: true}, domain.StepBusinessProfile, true},
		{"ferretería: sin movimientos de stock el inventario inicial queda pendiente", hardwareSteps, domain.Facts{HasProduct: true}, domain.StepInitialStock, false},
		{"ferretería: con un movimiento de stock el inventario inicial está completo", hardwareSteps, domain.Facts{HasStock: true}, domain.StepInitialStock, true},
		{"ferretería: con otro usuario o una invitación el equipo está completo", hardwareSteps, domain.Facts{HasTeammate: true}, domain.StepInviteTeam, true},
		{"ferretería: un cliente con cupo completa el de crédito", hardwareSteps, domain.Facts{HasCreditCustomer: true}, domain.StepCustomerCredit, true},
		{"ferretería: con la emisión activa sin resolución vigente la facturación queda pendiente", hardwareSteps, domain.Facts{InvoicingEnabled: true}, domain.StepInvoicing, false},
		{"ropa: con la emisión activa y una resolución vigente la facturación está completa", clothingSteps, domain.Facts{InvoicingEnabled: true, InvoicingReady: true}, domain.StepInvoicing, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := domain.BuildChecklist(domain.ChecklistInput{
				Steps: tc.steps, Facts: tc.facts, Can: owner, Modules: []string{"unit_conversions", "variants"},
			})
			require.Equal(t, tc.done, stepDone(t, c, tc.step))
		})
	}
}

func TestBuildChecklist_StepsFromTheTemplate(t *testing.T) {
	modules := []string{"unit_conversions", "variants"}
	facts := domain.Facts{InvoicingEnabled: true}

	t.Run("dos plantillas distintas producen listas de pasos distintas", func(t *testing.T) {
		hardware := domain.BuildChecklist(domain.ChecklistInput{Steps: hardwareSteps, Facts: facts, Can: owner, Modules: modules})
		clothing := domain.BuildChecklist(domain.ChecklistInput{Steps: clothingSteps, Facts: facts, Can: owner, Modules: modules})
		require.Contains(t, stepIDs(hardware), domain.StepAlternateUnits)
		require.NotContains(t, stepIDs(clothing), domain.StepAlternateUnits)
		require.Contains(t, stepIDs(clothing), domain.StepVariantProducts)
		require.NotEqual(t, stepIDs(hardware), stepIDs(clothing))
	})
	t.Run("respeta el orden de la plantilla", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{Steps: clothingSteps, Facts: facts, Can: owner, Modules: modules})
		require.Equal(t, []domain.StepID{
			domain.StepBusinessProfile, domain.StepInvoicing, domain.StepProducts, domain.StepVariantProducts,
			domain.StepInitialStock, domain.StepInviteTeam, domain.StepFirstSale,
		}, stepIDs(c))
	})
	t.Run("un paso de un módulo inactivo no aparece", func(t *testing.T) {
		hardware := domain.BuildChecklist(domain.ChecklistInput{Steps: hardwareSteps, Facts: facts, Can: owner})
		clothing := domain.BuildChecklist(domain.ChecklistInput{Steps: clothingSteps, Facts: facts, Can: owner})
		require.NotContains(t, stepIDs(hardware), domain.StepAlternateUnits)
		require.NotContains(t, stepIDs(clothing), domain.StepVariantProducts)
	})
	t.Run("el paso de facturación solo aparece con la emisión activa", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{Steps: hardwareSteps, Can: owner, Modules: modules})
		require.NotContains(t, stepIDs(c), domain.StepInvoicing)
	})
	t.Run("el paso de facturación es crítico", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{Steps: hardwareSteps, Facts: facts, Can: owner, Modules: modules})
		for _, s := range c.Steps {
			require.Equal(t, s.ID == domain.StepInvoicing, s.Critical, s.ID)
		}
	})
}

func TestBuildChecklist_WhoSeesIt(t *testing.T) {
	modules := []string{"unit_conversions", "variants"}

	t.Run("cada rol ve solo los pasos que puede resolver", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{
			Steps: hardwareSteps, Facts: domain.Facts{InvoicingEnabled: true}, Modules: modules,
			Can: only("catalog:write", "inventory:adjust", "sales:create"),
		})
		require.Equal(t, []domain.StepID{
			domain.StepProducts, domain.StepAlternateUnits, domain.StepInitialStock, domain.StepFirstSale,
		}, stepIDs(c))
		require.Equal(t, 4, c.Total())
	})
	t.Run("no se muestra a un rol sin permiso para resolver ningún paso", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{Steps: clothingSteps, Modules: modules, Can: only("catalog:read")})
		require.Empty(t, c.Steps)
		require.False(t, c.Visible())
	})
	t.Run("muestra el progreso de lo completado", func(t *testing.T) {
		c := domain.BuildChecklist(domain.ChecklistInput{
			Steps: clothingSteps, Modules: modules, Can: owner,
			Facts: domain.Facts{HasContact: true, HasProduct: true},
		})
		require.Equal(t, 2, c.Completed())
		require.Equal(t, 6, c.Total())
		require.True(t, c.Visible())
	})
	t.Run("la lista desaparece al completar todos los pasos", func(t *testing.T) {
		for _, steps := range [][]string{hardwareSteps, clothingSteps} {
			c := domain.BuildChecklist(domain.ChecklistInput{Steps: steps, Facts: allDone, Modules: modules, Can: owner})
			require.Equal(t, c.Total(), c.Completed())
			require.False(t, c.Visible())
		}
	})
}
