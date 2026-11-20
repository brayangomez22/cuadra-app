//go:build integration

package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// onboardingChecklist is the body of GET /onboarding/checklist.
type onboardingChecklist struct {
	Visible, Dismissed bool
	Completed, Total   int
	Steps              []struct {
		ID             string `json:"id"`
		Done, Critical bool
	}
}

func (c onboardingChecklist) ids() []string {
	ids := make([]string, 0, len(c.Steps))
	for _, s := range c.Steps {
		ids = append(ids, s.ID)
	}
	return ids
}

// done lists the steps that are done.
func (c onboardingChecklist) done() []string {
	var ids []string
	for _, s := range c.Steps {
		if s.Done {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

func checklistOf(t *testing.T, f *salesFlow, token string) onboardingChecklist {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/onboarding/checklist", "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var c onboardingChecklist
	decode(t, rec, &c)
	return c
}

// TestOnboardingChecklist runs T44 through the whole stack: the steps come
// from the template and each one is done only once its data exists.
func TestOnboardingChecklist(t *testing.T) {
	t.Run("una empresa nueva ve sus primeros pasos y se completan con datos reales", func(t *testing.T) {
		fresh := &salesFlow{}
		fresh.handler, fresh.tokens = newAPI(t)
		owner, _ := fresh.signup(t, "ana@nueva.co")

		c := checklistOf(t, fresh, owner)
		require.True(t, c.Visible)
		require.Equal(t, []string{"business_profile", "products", "alternate_units", "initial_stock", "customer_credit", "invite_team", "first_sale"}, c.ids(),
			"los de la plantilla de ferretería, sin la facturación porque la emisión está apagada")
		require.Empty(t, c.done())

		f := newReportsFlowOf(t, newSalesFlowOf(t, "eva@primeros.co")) // staff, a product, a location and a receipt; the drawer open
		require.Equal(t, []string{"products", "initial_stock", "invite_team"}, checklistOf(t, f.salesFlow, f.ownerToken).done())

		f.sell(t, "1", "35700")
		rec := f.do(t, http.MethodPatch, "/api/v1/tenant", `{"address":"Calle 45 # 12-30","city":"Medellín","phone":"604 444 1234"}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		c = checklistOf(t, f.salesFlow, f.ownerToken)
		require.Equal(t, []string{"business_profile", "products", "initial_stock", "invite_team", "first_sale"}, c.done())
		require.Equal(t, 5, c.Completed)
		require.Equal(t, 7, c.Total)
	})

	t.Run("cada rol ve solo los pasos que puede resolver", func(t *testing.T) {
		f := newSalesFlowOf(t, "rosa@primeros.co")

		require.Equal(t, []string{"first_sale"}, checklistOf(t, f, f.cashierToken).ids())
		require.Equal(t, []string{"initial_stock"}, checklistOf(t, f, f.warehouseToken).ids())
	})

	t.Run("oculta la lista y vuelve si la facturación queda sin resolución", func(t *testing.T) {
		f := newSalesFlowOf(t, "marta@primeros.co")
		rec := f.do(t, http.MethodPut, "/api/v1/onboarding/checklist/dismissal", "", f.ownerToken)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		c := checklistOf(t, f, f.ownerToken)
		require.True(t, c.Dismissed)
		require.False(t, c.Visible)

		rec = f.do(t, http.MethodPut, "/api/v1/invoicing/settings", `{"enabled":true,"default_kind":"pos"}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		c = checklistOf(t, f, f.ownerToken)
		require.Contains(t, c.ids(), "invoicing")
		require.True(t, c.Visible, "falta un paso crítico que no faltaba al ocultarla")

		rec = f.do(t, http.MethodPost, "/api/v1/invoicing/resolutions", `{"kind":"pos","prefix":"pos","number":"18764000001234","range_from":1,"range_to":10,`+
			`"valid_from":"2026-01-01","valid_to":"`+time.Now().AddDate(1, 0, 0).Format(time.DateOnly)+`"}`, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, checklistOf(t, f, f.ownerToken).done(), "invoicing")
	})

	t.Run("aislamiento: los datos de una empresa no completan los pasos de otra", func(t *testing.T) {
		f := newSalesFlowOf(t, "lina@primeros.co")
		other, _ := f.signup(t, "luis@otra.co")

		require.Contains(t, checklistOf(t, f, f.ownerToken).done(), "products")
		require.Empty(t, checklistOf(t, f, other).done())
	})

	t.Run("el registro con consentimiento guarda la analítica", func(t *testing.T) {
		f := &salesFlow{}
		f.handler, f.tokens = newAPI(t)
		rec := f.do(t, http.MethodPost, "/api/v1/auth/signup",
			`{"tenant_name":"Calzado Andino","nit":"900123456-8","name":"Ana Gómez","email":"ana@andino.co","password":"clave-segura-123","business_template":"ropa_calzado","analytics_consent":true}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var session struct {
			AccessToken string `json:"access_token"`
		}
		decode(t, rec, &session)

		rec = f.do(t, http.MethodGet, "/api/v1/me", "", session.AccessToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var me struct {
			Consent bool `json:"tenant_analytics_consent"`
		}
		decode(t, rec, &me)
		require.True(t, me.Consent)
	})
}
