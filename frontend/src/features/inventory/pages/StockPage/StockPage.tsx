import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { NoLocations } from './components/NoLocations/NoLocations';
import { StockActions } from './components/StockActions/StockActions';
import { StockBrowser } from './components/StockBrowser/StockBrowser';
import { useStockLocation } from '../../hooks/useStockLocation';
import styles from './StockPage.module.scss';

/** /inventario: the stock of one location (sede), chosen in the URL (?sede=). */
export function StockPage() {
  const can = useCan();
  const locations = useLocations();
  const list = locations.data ?? [];
  const { location, chooseLocation, offset, setOffset } = useStockLocation(list);
  const canManage = can('tenant:manage');
  const canReceive = can('inventory:adjust');

  return (
    <Page
      title="Inventario"
      description="Lo que hay en cada sede, en la unidad de cada producto."
      actions={
        <StockActions
          location={location}
          canReceive={canReceive}
          canCreate={canManage && list.length > 0}
          onCreated={chooseLocation}
        />
      }
    >
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={<Spinner label="Cargando sedes…" className={styles.loading} />}
        isEmpty={!location}
        empty={<NoLocations canManage={canManage} onCreated={chooseLocation} />}
      >
        {() =>
          location && (
            <StockBrowser
              locations={list}
              location={location}
              onLocationChange={chooseLocation}
              offset={offset}
              onOffsetChange={setOffset}
              canSeeCost={can('catalog:write')}
              canReceive={canReceive}
            />
          )
        }
      </QueryState>
    </Page>
  );
}
