import { QueryState } from '@/shared/query';
import { EmptyState, IconBox, Page, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { LocationReorder } from './components/LocationReorder/LocationReorder';
import { ReorderFilters } from './components/ReorderFilters/ReorderFilters';
import { useReorderParams } from './hooks/useReorderParams';
import styles from './ReorderPage.module.scss';

/** /inventario/reponer: what reached its minimum at one location or at all (?sede=todas), with how much to order. */
export function ReorderPage() {
  const locations = useLocations();
  const list = locations.data ?? [];
  const params = useReorderParams(list);
  const locationName = (id: string) => list.find((l) => l.id === id)?.name ?? '';

  return (
    <Page title="Por reponer" description="Lo que llegó a su stock mínimo, con cuánto pedir para llegar al máximo.">
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={<Spinner label="Cargando sedes…" className={styles.loading} />}
        isEmpty={list.length === 0}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Aún no hay sedes"
            description="Cree una sede en el inventario para llevar sus existencias y sus mínimos."
          />
        }
      >
        <ReorderFilters locations={list} params={params} />
        <LocationReorder
          location={params.location}
          locationName={params.all ? locationName : undefined}
          sort={params.sort}
          offset={params.offset}
          onOffsetChange={params.setOffset}
        />
      </QueryState>
    </Page>
  );
}
