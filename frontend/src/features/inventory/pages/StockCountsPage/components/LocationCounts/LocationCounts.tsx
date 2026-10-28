import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { EmptyState, IconClipboardCheck, Inline, Pagination, Spinner, Stack } from '@/shared/ui';
import { useStockCounts } from '../../../../api/stockCounts';
import { PAGE_SIZE, type Location } from '../../../../api/types';
import { CountsTable } from '../CountsTable/CountsTable';
import { OpenCount } from '../OpenCount/OpenCount';
import { StartCountDialog } from '../StartCountDialog/StartCountDialog';
import styles from './LocationCounts.module.scss';

type Props = { location: Location; offset: number; onOffsetChange: (offset: number) => void };

/** The open count of a location, if any, and a page of its counts. */
export function LocationCounts({ location, offset, onOffsetChange }: Props) {
  const counts = useStockCounts(location.id, offset);
  const manages = useCan()('inventory:count_manage');
  const items = counts.data?.items ?? [];
  const open = items.find((count) => count.status === 'open');
  const closed = items.filter((count) => count.status !== 'open');
  // A location has one open count at a time: the API rejects a second one.
  const start = manages && !open && <StartCountDialog location={location} />;

  return (
    <QueryState
      queries={[counts]}
      errorTitle="No pudimos cargar los conteos"
      loading={<Spinner label="Cargando conteos…" className={styles.loading} />}
      isEmpty={items.length === 0}
      empty={
        <EmptyState
          icon={<IconClipboardCheck />}
          title={`Aún no hay conteos en ${location.name}`}
          description="Un conteo compara lo que hay en el estante con lo que dice el sistema, mientras se sigue vendiendo."
          action={start}
        />
      }
    >
      {() => (
        <Stack gap="6" aria-busy={counts.isPlaceholderData || undefined}>
          {start && <Inline justify="end">{start}</Inline>}
          {open && <OpenCount count={open} canReview={manages} />}
          {closed.length > 0 && (
            <CountsTable caption={`Conteos de ${location.name}`} counts={closed} canReview={manages} />
          )}
          <Pagination
            label="Páginas de los conteos"
            total={counts.data?.total ?? 0}
            limit={PAGE_SIZE}
            offset={offset}
            onOffsetChange={onOffsetChange}
          />
        </Stack>
      )}
    </QueryState>
  );
}
