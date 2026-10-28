import { paths } from '@/app/routes/paths';
import { useLocations } from '@/features/inventory';
import { QueryState } from '@/shared/query';
import { ButtonLink, EmptyState, IconBox, Page, Spinner } from '@/shared/ui';
import { ImportWizard } from './components/ImportWizard/ImportWizard';
import styles from './ImportPage.module.scss';

/** /productos/importar: load the catalog and the initial stock from a spreadsheet. */
export function ImportPage() {
  const locations = useLocations();
  const options = (locations.data ?? [])
    .filter((location) => location.active)
    .map((location) => ({ value: location.id, label: location.name }));

  return (
    <Page
      title="Importar productos"
      description="Cargue su catálogo, sus precios y el stock inicial desde una hoja de cálculo."
    >
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={<Spinner label="Cargando…" className={styles.loading} />}
        isEmpty={options.length === 0}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Aún no hay sedes activas"
            description="El stock inicial entra a una sede o bodega: primero cree una."
            action={
              <ButtonLink to={paths.inventory.stock()} variant="secondary">
                Ir a inventario
              </ButtonLink>
            }
          />
        }
      >
        {() => <ImportWizard locations={options} />}
      </QueryState>
    </Page>
  );
}
