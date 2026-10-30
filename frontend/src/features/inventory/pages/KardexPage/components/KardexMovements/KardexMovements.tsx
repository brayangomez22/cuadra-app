import { QueryState } from '@/shared/query';
import { EmptyState, IconBox, Pagination, Spinner } from '@/shared/ui';
import type { useKardex } from '../../../../api/kardex';
import { PAGE_SIZE, type Location } from '../../../../api/types';
import type { useKardexFilters } from '../../hooks/useKardexFilters';
import { MovementsTable } from '../MovementsTable/MovementsTable';
import styles from './KardexMovements.module.scss';

type Props = {
  kardex: ReturnType<typeof useKardex>;
  caption: string;
  withAxes: boolean;
  filters: ReturnType<typeof useKardexFilters>;
  locations: Location[];
  canSeeCost: boolean;
};

/** A page of movements, newest first, or why there are none. */
export function KardexMovements({ kardex, caption, withAxes, filters, locations, canSeeCost }: Props) {
  const empty = (
    <EmptyState
      icon={<IconBox />}
      title={withAxes ? 'Esta variante aún no tiene movimientos' : 'Este producto aún no tiene movimientos'}
      description={
        filters.locationId
          ? 'No hay entradas, ventas ni ajustes en esta sede.'
          : 'Las entradas, ventas, ajustes y traslados aparecerán aquí.'
      }
    />
  );
  return (
    <QueryState
      queries={[kardex]}
      errorTitle="No pudimos cargar los movimientos"
      loading={<Spinner label="Cargando movimientos…" className={styles.loading} />}
      isEmpty={kardex.data?.items.length === 0}
      empty={empty}
    >
      {() => (
        <div className={styles.root} aria-busy={kardex.isPlaceholderData || undefined}>
          <MovementsTable
            caption={caption}
            movements={kardex.data?.items ?? []}
            locations={locations}
            allLocations={!filters.locationId}
            canSeeCost={canSeeCost}
          />
          <Pagination
            label="Páginas del kardex"
            total={kardex.data?.total ?? 0}
            limit={PAGE_SIZE}
            offset={filters.offset}
            onOffsetChange={filters.setOffset}
          />
        </div>
      )}
    </QueryState>
  );
}
