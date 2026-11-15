package app_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

type fakeAccounts struct{ *store }

func (f fakeAccounts) Get(_ context.Context, tenantID uuid.UUID) (domain.ProviderAccount, error) {
	if a, ok := f.accounts[tenantID]; ok {
		return a, nil
	}
	return domain.DefaultProviderAccount(), nil
}

func (f fakeAccounts) Save(_ context.Context, tenantID uuid.UUID, a domain.ProviderAccount) error {
	f.accounts[tenantID] = a
	return nil
}

type fakeFiles struct{ *store }

func (f fakeFiles) Put(_ context.Context, tenantID, documentID uuid.UUID, file domain.File) error {
	if f.filesDown {
		return errors.New("storage unavailable")
	}
	f.files[fileKey{tenantID, documentID, file.Kind}] = file
	return nil
}

func (f fakeFiles) Get(_ context.Context, tenantID, documentID uuid.UUID, kind domain.FileKind) (domain.File, error) {
	file, ok := f.files[fileKey{tenantID, documentID, kind}]
	if !ok {
		return domain.File{}, domain.ErrFileNotFound
	}
	return file, nil
}

// fakeIssuers is the company of each tenant; one it does not know is a real
// tenant without data.
type fakeIssuers struct{ issuers map[uuid.UUID]domain.Issuer }

func (f fakeIssuers) Issuer(_ context.Context, tenantID uuid.UUID) (domain.Issuer, error) {
	return f.issuers[tenantID], nil
}

// demo marks the tenant as a demonstration tenant.
func (f fakeIssuers) demo(tenantID uuid.UUID) {
	i := f.issuers[tenantID]
	i.Demo = true
	f.issuers[tenantID] = i
}
