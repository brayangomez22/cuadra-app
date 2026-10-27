import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Field, Page, Select, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { locationOptions } from '../../model/shownLocation';
import { LocationStock } from './components/LocationStock/LocationStock';
import { NoLocations } from './components/NoLocations/NoLocations';
import { StockActions } from './components/StockActions/StockActions';
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
            <>
              <Field label="Sede" className={styles.location}>
                <Select options={locationOptions(list)} value={location.id} onValueChange={chooseLocation} />
              </Field>
              <LocationStock
                location={location}
                offset={offset}
                canSeeCost={can('catalog:write')}
                canReceive={canReceive}
                onOffsetChange={setOffset}
              />
            </>
          )
        }
      </QueryState>
    </Page>
  );
}
