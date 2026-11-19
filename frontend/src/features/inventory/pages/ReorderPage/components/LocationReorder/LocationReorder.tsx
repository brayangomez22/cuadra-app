import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { Alert, ButtonLink, EmptyState, IconBox, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, type Location, type ReorderSort } from '../../../../api/types';
import { useReorderRows } from '../../hooks/useReorderRows';
import { ReorderTable } from '../ReorderTable/ReorderTable';
import styles from './LocationReorder.module.scss';

type Props = {
  /** The location; undefined lists every one. */
  location: Location | undefined;
  /** With every location, the name of each row's location. */
  locationName?: (id: string) => string;
  sort: ReorderSort;
  offset: number;
  onOffsetChange: (offset: number) => void;
};

/** A page of what to reorder at a location (or at all of them), or that there is nothing to. */
export function LocationReorder({ location, locationName, sort, offset, onOffsetChange }: Props) {
  const { alerts, rows, products, productsById } = useReorderRows({ locationId: location?.id, offset, sort });
  const where = location ? `en ${location.name}` : 'en todas las sedes';
  const empty = (
    <EmptyState
      icon={<IconBox />}
      title={`Nada por reponer ${where}`}
      description="Aquí aparece cada producto que llegue a su stock mínimo. El mínimo y el máximo se definen en el inventario."
      action={
        <ButtonLink to={paths.inventory.stock({ sede: location?.id })} variant="secondary">
          Ver inventario
        </ButtonLink>
      }
    />
  );

  return (
    <QueryState
      queries={[alerts]}
      errorTitle="No pudimos cargar lo que hay por reponer"
      loading={<Spinner label="Cargando…" className={styles.loading} />}
      isEmpty={rows.length === 0}
      empty={empty}
    >
      {() => (
        <div className={styles.root} aria-busy={alerts.isPlaceholderData || undefined}>
          {products.isError && (
            <Alert tone="warning">
              No pudimos cargar los nombres de los productos. Las cantidades sí están al día.
            </Alert>
          )}
          <ReorderTable
            caption={`Por reponer ${where}`}
            locationName={locationName}
            alerts={rows}
            productsById={productsById}
            loadingProducts={products.isPending}
          />
          <Pagination
            label="Páginas de lo que hay por reponer"
            total={alerts.data?.total ?? 0}
            limit={PAGE_SIZE}
            offset={offset}
            onOffsetChange={onOffsetChange}
          />
        </div>
      )}
    </QueryState>
  );
}
