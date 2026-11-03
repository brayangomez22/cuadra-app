import { useAuth } from '@/features/auth';
import { useLocations } from '@/features/inventory';
import { QueryState } from '@/shared/query';
import { EmptyState, IconCash, Page, Pagination, Spinner, Stack } from '@/shared/ui';
import { CASH_HISTORY_PAGE_SIZE, useCashSessions, useOpenCashSessions } from '../../api/cashSessions';
import { CashHistoryFilters } from './components/CashHistoryFilters/CashHistoryFilters';
import { CashHistoryTable } from './components/CashHistoryTable/CashHistoryTable';
import { OpenDrawers } from './components/OpenDrawers/OpenDrawers';
import { useCashHistoryParams } from './hooks/useCashHistoryParams';

/** /caja/cierres: the drawers open now, then every drawer of the business with its cashier, what it sold and how its count came out. */
export function CashHistoryPage() {
  const params = useCashHistoryParams();
  const sessions = useCashSessions(params.query);
  // Whatever the filters, the drawers open now come first.
  const open = (useOpenCashSessions().data?.items ?? []).filter((s) => s.status === 'open');
  const locations = useLocations().data ?? [];
  const auth = useAuth();
  const myId = auth.status === 'authenticated' ? auth.user.id : undefined;
  const locationName = (id: string) => locations.find((l) => l.id === id)?.name ?? '';
  return (
    <Page
      title="Cierres de caja"
      description="Las cajas de todos los cajeros: abra una para ver su cierre o cierre la que alguien dejó abierta."
    >
      <OpenDrawers sessions={open} locationName={locationName} myId={myId} />
      <CashHistoryFilters params={params} locations={locations} />
      <QueryState
        queries={[sessions]}
        errorTitle="No pudimos cargar las cajas"
        loading={<Spinner label="Cargando cajas…" />}
        isEmpty={sessions.data?.items.length === 0}
        empty={<EmptyState icon={<IconCash />} title="No hay cajas con ese filtro" />}
      >
        {() =>
          sessions.data && (
            <Stack gap="4" aria-busy={sessions.isPlaceholderData || undefined}>
              <CashHistoryTable sessions={sessions.data.items} locationName={locationName} />
              <Pagination
                label="Páginas de cajas"
                total={sessions.data.total}
                limit={CASH_HISTORY_PAGE_SIZE}
                offset={params.offset}
                onOffsetChange={params.setOffset}
              />
            </Stack>
          )
        }
      </QueryState>
    </Page>
  );
}
