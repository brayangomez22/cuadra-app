import { useLocations } from '@/features/inventory';
import { QueryState } from '@/shared/query';
import { EmptyState, IconBox, Page, Spinner } from '@/shared/ui';
import { useCurrentCashSession } from '../../api/cashSessions';
import styles from './CashPage.module.scss';
import { OpenCashForm } from './components/OpenCashForm/OpenCashForm';
import { OpenSession } from './components/OpenSession/OpenSession';

const title = 'Caja';

/**
 * /caja: the cashier's drawer. Without one open, the form to open it with
 * the float; with one open, its float and cash ins and outs, and the way to
 * close it. The expected cash is never shown here: it comes after the count.
 */
export function CashPage() {
  const current = useCurrentCashSession();
  const locations = useLocations();

  if (!current.isSuccess || !locations.isSuccess) {
    return (
      <Page title={title}>
        <QueryState
          queries={[current, locations]}
          errorTitle="No pudimos cargar la caja"
          loading={<Spinner label="Cargando la caja…" className={styles.loading} />}
        >
          {null}
        </QueryState>
      </Page>
    );
  }

  const session = current.data.session;
  if (session) {
    const location = locations.data.find((l) => l.id === session.location_id);
    return <OpenSession session={session} locationName={location?.name ?? ''} />;
  }

  const active = locations.data.filter((l) => l.active);
  if (active.length === 0) {
    return (
      <Page title={title}>
        <EmptyState
          icon={<IconBox />}
          title="No hay sedes activas"
          description="La caja se abre en una sede. Pida al dueño que cree o active una en Inventario."
        />
      </Page>
    );
  }
  return (
    <Page title={title} description="Cuente el efectivo con el que empieza y abra la caja para vender.">
      <OpenCashForm locations={active.map((l) => ({ value: l.id, label: l.name }))} />
    </Page>
  );
}
