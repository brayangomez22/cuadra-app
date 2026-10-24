import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { ButtonLink, EmptyState, IconBox, Page, Spinner } from '@/shared/ui';
import { ReceiptForm } from './components/ReceiptForm/ReceiptForm';
import { useReceiptSetup } from './hooks/useReceiptSetup';
import styles from './ReceiptPage.module.scss';

/** /inventario/entrada: goods that arrive at a location. */
export function ReceiptPage() {
  const { locations, presetPending, options, initialItem, initialLocation } = useReceiptSetup();
  const loading = <Spinner label="Cargando…" className={styles.loading} />;

  return (
    <Page
      title="Registrar entrada"
      description="La mercancía que llega suma a la existencia de la sede y actualiza el costo promedio."
    >
      <QueryState
        queries={[locations]}
        errorTitle="No pudimos cargar las sedes"
        loading={loading}
        isEmpty={options.length === 0}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Aún no hay sedes activas"
            description="Para registrar mercancía primero debe existir una sede o bodega."
            action={
              <ButtonLink to={paths.inventory.stock()} variant="secondary">
                Ir a inventario
              </ButtonLink>
            }
          />
        }
      >
        {() =>
          presetPending ? (
            loading
          ) : (
            <ReceiptForm locations={options} initialItem={initialItem} initialLocation={initialLocation} />
          )
        }
      </QueryState>
    </Page>
  );
}
