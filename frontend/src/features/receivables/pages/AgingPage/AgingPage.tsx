import { formatDay } from '@/shared/format';
import { pageParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { EmptyState, IconSuccess, Page, Pagination } from '@/shared/ui';
import { PAGE_SIZE, useAging } from '../../api/aging';
import { AgingTable } from './components/AgingTable/AgingTable';
import { AgingTotals } from './components/AgingTotals/AgingTotals';
import styles from './AgingPage.module.scss';

/** The receivables by age at /cartera: what each customer owes, due and overdue, most overdue first. */
export function AgingPage() {
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAGE_SIZE;
  const aging = useAging({ limit: PAGE_SIZE, offset });
  return (
    <Page
      title="Cartera"
      description={
        aging.data
          ? `Lo que le deben los clientes por edad, contada desde el vencimiento al ${formatDay(aging.data.as_of)}.`
          : 'Lo que le deben los clientes por edad, contada desde el vencimiento.'
      }
    >
      <QueryState
        queries={[aging]}
        errorTitle="No pudimos cargar la cartera"
        isEmpty={aging.data?.items.length === 0}
        empty={
          <EmptyState
            icon={<IconSuccess />}
            title="Nadie debe a la empresa"
            description="Cuando venda a crédito, aquí verá lo que le deben y desde cuándo."
          />
        }
      >
        {() =>
          aging.data && (
            <div className={styles.root} aria-busy={aging.isPlaceholderData || undefined}>
              <AgingTotals totals={aging.data.totals} />
              <AgingTable rows={aging.data.items} totals={aging.data.totals} />
              <Pagination
                label="Páginas de la cartera"
                total={aging.data.total}
                limit={PAGE_SIZE}
                offset={offset}
                onOffsetChange={(next) => setPage(next / PAGE_SIZE + 1)}
              />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
