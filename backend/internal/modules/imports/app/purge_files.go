package app

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// purgeBatch bounds the files one run deletes; the daily job drains a
// backlog over several nights.
const purgeBatch = 1000

// PurgeFiles deletes the files of the imports that ended more than
// domain.FileRetention ago, in every tenant, and returns how many. Each
// import is read and checked within its tenant before its file goes.
func (s *Service) PurgeFiles(ctx context.Context) (_ int, err error) {
	ctx, span := s.spans.Start(ctx, "PurgeFiles")
	defer func() { usecase.End(span, err) }()

	now := s.Now()
	refs, err := s.Imports.PurgeableFiles(ctx, now.Add(-domain.FileRetention), purgeBatch)
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, ref := range refs {
		imp, err := s.Imports.Get(ctx, ref.TenantID, ref.ImportID)
		if err != nil {
			return purged, err
		}
		if !imp.FileExpired(now) {
			continue
		}
		if err := s.Imports.DropFile(ctx, ref.TenantID, ref.ImportID); err != nil {
			return purged, err
		}
		purged++
		s.filesPurged.Add(ctx, 1)
	}
	span.SetAttributes(attribute.Int("imports.files_purged", purged))
	s.Logger.InfoContext(ctx, "import files purged", slog.Int("files", purged))
	return purged, nil
}
