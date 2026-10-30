import { QueryState } from '@/shared/query';
import { Alert, EmptyState, IconSearch, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, type Location } from '../../../../api/types';
import { EmptyStock } from '../EmptyStock/EmptyStock';
import { useStockRows } from '../../hooks/useStockRows';
import { StockTable } from '../StockTable/StockTable';
import styles from './LocationStock.module.scss';

type Props = {
  location: Location;
  offset: number;
  /** What the user searched (name, SKU or barcode); empty for the whole location. */
  term: string;
  canSeeCost: boolean;
  canReceive: boolean;
  onOffsetChange: (offset: number) => void;
};

/** A page of a location's stock (or of what a search found), or why there is nothing to show. */
export function LocationStock({ location, offset, term, canSeeCost, canReceive, onOffsetChange }: Props) {
  const { stock, found, levels, products, productsById, noMatch } = useStockRows(location.id, offset, term);
  const nothing = term ? (
    <EmptyState
      icon={<IconSearch />}
      title={noMatch ? `Ningún producto coincide con «${term}»` : `Nada de «${term}» en ${location.name}`}
      description={
        noMatch ? 'Busque por nombre, SKU o código de barras.' : 'Lo que coincide no tiene movimientos en esta sede.'
      }
    />
  ) : (
    <EmptyStock location={location} canReceive={canReceive} />
  );

  return (
    <QueryState
      // Without a match the stock is never asked for: only the search counts.
      queries={term ? (noMatch ? [found] : [found, stock]) : [stock]}
      errorTitle="No pudimos cargar las existencias"
      loading={<Spinner label="Cargando existencias…" className={styles.loading} />}
      isEmpty={noMatch || levels.length === 0}
      empty={nothing}
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
