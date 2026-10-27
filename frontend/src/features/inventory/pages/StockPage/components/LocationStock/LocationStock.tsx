import { QueryState } from '@/shared/query';
import { Alert, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, type Location } from '../../../../api/types';
import { EmptyStock } from '../EmptyStock/EmptyStock';
import { useStockRows } from '../../hooks/useStockRows';
import { StockTable } from '../StockTable/StockTable';
import styles from './LocationStock.module.scss';

type Props = {
  location: Location;
  offset: number;
  canSeeCost: boolean;
  canReceive: boolean;
  onOffsetChange: (offset: number) => void;
};

/** A page of a location's stock, or why there is nothing to show. */
export function LocationStock({ location, offset, canSeeCost, canReceive, onOffsetChange }: Props) {
  const { stock, levels, products, productsById } = useStockRows(location.id, offset);

  return (
    <QueryState
      queries={[stock]}
      errorTitle="No pudimos cargar las existencias"
      loading={<Spinner label="Cargando existencias…" className={styles.loading} />}
      isEmpty={levels.length === 0}
      empty={<EmptyStock location={location} canReceive={canReceive} />}
    >
      {() => (
        <div className={styles.root} aria-busy={stock.isPlaceholderData || undefined}>
          {products.isError && (
            <Alert tone="warning">
              No pudimos cargar los nombres de los productos. Las cantidades sí están al día.
            </Alert>
          )}
          <StockTable
            caption={`Existencias en ${location.name}`}
            levels={levels}
            productsById={productsById}
            loadingProducts={products.isPending}
            canSeeCost={canSeeCost}
            canEditReorder={canReceive}
          />
          <Pagination
            label="Páginas de existencias"
            total={stock.data?.total ?? 0}
            limit={PAGE_SIZE}
            offset={offset}
            onOffsetChange={onOffsetChange}
          />
        </div>
      )}
    </QueryState>
  );
}
