import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Field, Page, Pagination, Select, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { useStockCount } from '../../api/stockCounts';
import { PAGE_SIZE } from '../../api/types';
import { lineFilterOptions, type LineFilter } from '../../model/stockCount/lineFilter';
import { CloseCountActions } from './components/CloseCountActions/CloseCountActions';
import { LinesTable } from './components/LinesTable/LinesTable';
import { ReviewSummary } from './components/ReviewSummary/ReviewSummary';
import { useCountReview } from './hooks/useCountReview';
import styles from './CountReviewPage.module.scss';

/**
 * /inventario/conteos/:countId/diferencias: what was expected and what was
 * counted of each product, to recount the doubtful ones and approve the count.
 */
export function CountReviewPage() {
  const { countId = '' } = useParams();
  const count = useStockCount(countId);
  const location = useLocations().data?.find((l) => l.id === count.data?.location_id);
  const review = useCountReview(countId);
  const open = count.data?.status === 'open';

  return (
    <Page
      title="Diferencias del conteo"
      description={location && `Conteo de ${location.name}`}
      actions={open && <CloseCountActions countId={countId} nameOf={review.nameOf} onRecounted={review.showRecount} />}
    >
      <QueryState
        queries={[count, review.lines]}
        errorTitle="No pudimos cargar las diferencias"
        loading={<Spinner label="Cargando las diferencias…" className={styles.loading} />}
      >
        {() =>
          review.lines.data && (
            <div className={styles.root} aria-busy={review.lines.isPlaceholderData || undefined}>
              <ReviewSummary summary={review.lines.data.summary} />
              <Field label="Mostrar" className={styles.filter}>
                <Select
                  options={lineFilterOptions}
                  value={review.filter}
                  onValueChange={(value) => review.setFilter(value as LineFilter)}
                />
              </Field>
              <LinesTable
                countId={countId}
                lines={review.rows}
                productsById={review.productsById}
                loadingProducts={review.products.isPending}
                canRecount={open}
                marked={review.marked}
              />
              <Pagination
                label="Páginas de las diferencias"
                total={review.lines.data.total}
                limit={PAGE_SIZE}
                offset={review.offset}
                onOffsetChange={review.setOffset}
              />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
