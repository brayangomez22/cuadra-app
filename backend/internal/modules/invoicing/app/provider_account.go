package app

import (
	"context"
	"log/slog"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ProviderAccountInput is a change of the tenant's account at the provider.
// A nil Token keeps the one stored.
type ProviderAccountInput struct {
	Mode      string
	TestSetID string
	Token     *string
}

// ProviderAccount returns the tenant's account at the provider. The API
// says whether it has a token, never the token.
func (s *Service) ProviderAccount(ctx context.Context, a Actor) (_ domain.ProviderAccount, err error) {
	ctx, span := s.startSpan(ctx, "GetProviderAccount", a)
	defer func() { usecase.End(span, err) }()

	return s.Accounts.Get(ctx, a.TenantID)
}

// UpdateProviderAccount sets the tenant's mode, test set and, if given, its
// organization's token. An invalid value changes nothing.
func (s *Service) UpdateProviderAccount(ctx context.Context, a Actor, in ProviderAccountInput) (_ domain.ProviderAccount, err error) {
	ctx, span := s.startSpan(ctx, "UpdateProviderAccount", a)
	defer func() { usecase.End(span, err) }()

	current, err := s.Accounts.Get(ctx, a.TenantID)
	if err != nil {
		return domain.ProviderAccount{}, err
	}
	token := current.Token
	if in.Token != nil {
		if token, err = domain.NewSecret(*in.Token); err != nil {
			return domain.ProviderAccount{}, err
		}
	}
	account, err := domain.NewProviderAccount(in.Mode, in.TestSetID, token)
	if err != nil {
		return domain.ProviderAccount{}, err
	}
	if err := s.Accounts.Save(ctx, a.TenantID, account); err != nil {
		return domain.ProviderAccount{}, err
	}
	s.Logger.InfoContext(ctx, "provider account updated",
		slog.String("mode", string(account.Mode)), slog.Bool("has_token", account.HasToken()),
		slog.Bool("token_changed", in.Token != nil))
	return account, nil
}
