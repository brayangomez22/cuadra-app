import { useState } from 'react';
import { Navigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useCurrentCashSession, useRefreshCurrentCash } from '../../api/cashSessions';
import styles from './CloseCashPage.module.scss';
import { CountForm } from './components/CountForm/CountForm';

const title = 'Cerrar caja';
const description =
  'Cuente el efectivo del cajón por billetes y monedas. Al cerrar, Cuadra lo compara con lo que debería haber.';

/**
 * /caja/cerrar: the count of the drawer (arqueo). It is blind: the screen
 * adds up what was counted, and the expected cash only shows in the report,
 * after closing.
 */
export function CloseCashPage() {
  const current = useCurrentCashSession();
  const refreshCurrentCash = useRefreshCurrentCash();
  const [closedId, setClosedId] = useState<string>();

  // Once closed, the user has no open session: go to the report, not to /caja.
  if (closedId) return <Navigate to={paths.sales.cashReport(closedId)} replace />;
  if (!current.isSuccess) {
    return (
      <Page title={title}>
        <QueryState
          queries={[current]}
          errorTitle="No pudimos cargar la caja"
          loading={<Spinner label="Cargando la caja…" className={styles.loading} />}
        >
          {null}
        </QueryState>
      </Page>
    );
  }
  const session = current.data.session;
  if (!session) return <Navigate to={paths.sales.cash()} replace />;
  return (
    <Page title={title} description={description}>
      <CountForm
        session={session}
        onClosed={(id) => {
          setClosedId(id);
          refreshCurrentCash();
        }}
      />
    </Page>
  );
}
