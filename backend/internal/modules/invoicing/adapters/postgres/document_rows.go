package postgres

import (
	"math"
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func addParams(s domain.DocumentSnapshot) sqlcgen.AddDocumentParams {
	return sqlcgen.AddDocumentParams{
		ID: s.ID, TenantID: s.TenantID, SaleID: s.SaleID, CreatedBy: s.CreatedBy, Kind: string(s.Kind),
		ResolutionID: s.ResolutionID, Prefix: s.Prefix, Number: s.Number, IssuedAt: s.IssuedAt,
		Status: string(s.Status), Attempts: attempts(s.Attempts),
		ProviderID: s.ProviderID, Code: s.Code, Qr: s.QR, Reasons: reasons(s.Reasons), LastError: s.LastError,
		AcceptedAt: s.AcceptedAt, UpdatedAt: s.UpdatedAt,
	}
}

func saveParams(s domain.DocumentSnapshot) sqlcgen.SaveDocumentParams {
	return sqlcgen.SaveDocumentParams{
		TenantID: s.TenantID, ID: s.ID, Status: string(s.Status),
		Attempts: attempts(s.Attempts), ProviderID: s.ProviderID, Code: s.Code, Qr: s.QR, Reasons: reasons(s.Reasons), LastError: s.LastError,
		AcceptedAt: s.AcceptedAt, UpdatedAt: s.UpdatedAt,
	}
}

// attempts converts the attempts to the column's int4; River gives up long
// before the bound.
func attempts(n int) int32 {
	if n < 0 || n > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(n)
}

// reasons never stores NULL: the column is NOT NULL.
func reasons(r []string) []string {
	if r == nil {
		return []string{}
	}
	return r
}

// documentOf rebuilds a stored document, in UTC and without an empty list
// of reasons, as the domain builds it.
func documentOf(row sqlcgen.ElectronicDocument) *domain.Document {
	var acceptedAt *time.Time
	if row.AcceptedAt != nil {
		at := row.AcceptedAt.UTC()
		acceptedAt = &at
	}
	var reasons []string
	if len(row.Reasons) > 0 {
		reasons = row.Reasons
	}
	return domain.RestoreDocument(domain.DocumentSnapshot{
		ID: row.ID, TenantID: row.TenantID, SaleID: row.SaleID, CreatedBy: row.CreatedBy,
		Kind: domain.Kind(row.Kind), ResolutionID: row.ResolutionID, Prefix: row.Prefix, Number: row.Number,
		IssuedAt: row.IssuedAt.UTC(), Status: domain.Status(row.Status), Attempts: int(row.Attempts),
		ProviderID: row.ProviderID, Code: row.Code, QR: row.Qr, Reasons: reasons, LastError: row.LastError,
		AcceptedAt: acceptedAt, UpdatedAt: row.UpdatedAt.UTC(),
	})
}

func documentsOf(rows []sqlcgen.ElectronicDocument) []*domain.Document {
	out := make([]*domain.Document, len(rows))
	for i, row := range rows {
		out[i] = documentOf(row)
	}
	return out
}
