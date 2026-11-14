// Package simulated is an ElectronicInvoicer that never reaches a provider
// nor the DIAN: it accepts every document with a made up CUDE. It is the
// provider of local development and tests, and of the demo tenant (P108,
// T36b), whose documents must never be real.
package simulated

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// Invoicer accepts every document at once. Its answer depends only on the
// document's number, so a look up finds what an issue accepted.
type Invoicer struct{}

// Issue accepts the document.
func (Invoicer) Issue(_ context.Context, req app.IssueRequest) (app.Result, error) {
	return accepted(req.DocumentRef), nil
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
