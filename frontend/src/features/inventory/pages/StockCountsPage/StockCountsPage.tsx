import { QueryState } from '@/shared/query';
import { EmptyState, Field, IconBox, Page, Select, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { useStockLocation } from '../../hooks/useStockLocation';
import { locationOptions } from '../../model/shownLocation';
import { LocationCounts } from './components/LocationCounts/LocationCounts';
import styles from './StockCountsPage.module.scss';

/** /inventario/conteos: the open count of a location (?sede=) and its past counts with their accuracy. */
export function StockCountsPage() {
  const locations = useLocations();
  const list = locations.data ?? [];
  const { location, chooseLocation, offset, setOffset } = useStockLocation(list);

  return (
    <Page
      title="Toma física"
      description="Cuente lo que hay en el estante y ajuste el inventario sin cerrar el negocio."
    >
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={<Spinner label="Cargando sedes…" className={styles.loading} />}
        isEmpty={!location}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Aún no hay sedes"
            description="Cree una sede en el inventario para poder contar sus existencias."
          />
        }
      >
        {() =>
          location && (
            <>
              <Field label="Sede" className={styles.location}>
                <Select options={locationOptions(list)} value={location.id} onValueChange={chooseLocation} />
              </Field>
              <LocationCounts location={location} offset={offset} onOffsetChange={setOffset} />
            </>
          )
        }
      </QueryState>
    </Page>
  );
}
