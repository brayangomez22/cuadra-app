// Package simulated is an ElectronicInvoicer that never reaches a provider
// nor the DIAN: it accepts every complete document with a made up CUDE, and
// refuses, as a provider does, one without the data the DIAN requires. It is the
// provider of local development and tests, and of the demo tenant (P108,
// T36b), whose documents must never be real.
package simulated

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// Invoicer accepts every complete document at once. Its answer depends
// only on the document's number, so a look up finds what an issue accepted.
type Invoicer struct{}

// Issue accepts the document, or refuses it (a bad request, which retrying
// does not fix) if it lacks what every document carries.
func (Invoicer) Issue(_ context.Context, req app.IssueRequest) (app.Result, error) {
	if err := incomplete(req); err != nil {
		return app.Result{}, &app.ProviderError{Reason: app.ReasonBadRequest, Err: err}
	}
	return accepted(req.DocumentRef), nil
}

// incomplete says what of the document's required data is missing: the
// buyer's DIAN document code and, of each line, its code, unit code and IVA
// treatment. It keeps Cuadra's own data honest without a provider.
func incomplete(req app.IssueRequest) error {
	if req.Buyer.IdentificationCode == "" {
		return errors.New("buyer without identification code")
	}
	if len(req.Sale.Lines) == 0 {
		return errors.New("document without lines")
	}
	for i, l := range req.Sale.Lines {
		if l.Code == "" || l.UnitCode == "" || l.TaxTreatment == "" {
			return fmt.Errorf("line %d without code, unit code or tax treatment", i+1)
		}
	}
	return nil
}

// Lookup finds every document: the simulation accepts them all.
func (Invoicer) Lookup(_ context.Context, ref app.DocumentRef) (app.Result, bool, error) {
	return accepted(ref), true, nil
}

// accepted is the simulated answer: a CUDE shaped like the DIAN's (SHA-384
// in hex) and a QR that says it has no legal value.
func accepted(ref app.DocumentRef) app.Result {
	key := fmt.Sprintf("%s/%s/%s/%d", ref.TenantID, ref.Kind, ref.Prefix, ref.Number)
	sum := sha512.Sum384([]byte(key))
	code := hex.EncodeToString(sum[:])
	return app.Result{
		Status:     domain.StatusAccepted,
		ProviderID: "simulado-" + code[:12],
		Code:       code,
		QR:         "Documento simulado, sin validez ante la DIAN. CUDE: " + code,
	}
}
