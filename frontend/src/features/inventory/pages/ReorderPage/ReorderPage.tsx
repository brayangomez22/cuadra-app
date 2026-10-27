import { QueryState } from '@/shared/query';
import { EmptyState, Field, IconBox, Page, Select, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { useStockLocation } from '../../hooks/useStockLocation';
import { locationOptions } from '../../model/shownLocation';
import { LocationReorder } from './components/LocationReorder/LocationReorder';
import styles from './ReorderPage.module.scss';

/** /inventario/reponer: what reached its minimum at one location (?sede=), with how much to order. */
export function ReorderPage() {
  const locations = useLocations();
  const list = locations.data ?? [];
  const { location, chooseLocation, offset, setOffset } = useStockLocation(list);

  return (
    <Page title="Por reponer" description="Lo que llegó a su stock mínimo, con cuánto pedir para llegar al máximo.">
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={<Spinner label="Cargando sedes…" className={styles.loading} />}
        isEmpty={!location}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Aún no hay sedes"
            description="Cree una sede en el inventario para llevar sus existencias y sus mínimos."
          />
        }
      >
        {() =>
          location && (
            <>
              <Field label="Sede" className={styles.location}>
                <Select options={locationOptions(list)} value={location.id} onValueChange={chooseLocation} />
              </Field>
              <LocationReorder location={location} offset={offset} onOffsetChange={setOffset} />
            </>
          )
        }
      </QueryState>
    </Page>
  );
}
