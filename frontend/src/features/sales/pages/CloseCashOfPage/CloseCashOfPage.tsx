import { useState } from 'react';
import { Navigate, useParams } from 'react-router';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useCashSession } from '../../api/cashSessions';
import { CountForm } from '../../components/CashCount/CountForm/CountForm';

const title = 'Cerrar la caja de otro usuario';
const description =
  'Cuente el efectivo del cajón que quedó abierto. El cierre queda a su nombre y Cuadra lo compara con lo esperado.';

/** /caja/:sessionId/cerrar: who manages cash counts a drawer another user left open. A closed one goes to its report. */
export function CloseCashOfPage() {
  const { sessionId = '' } = useParams();
  const session = useCashSession(sessionId);
  const [closed, setClosed] = useState(false);

  if (closed || session.data?.status === 'closed') return <Navigate to={paths.sales.cashReport(sessionId)} replace />;
  return (
    <Page title={title} description={description}>
      <QueryState
        queries={[session]}
        errorTitle="No pudimos cargar la caja"
        loading={<Spinner label="Cargando la caja…" />}
      >
        {() => session.data && <CountForm session={session.data} onClosed={() => setClosed(true)} />}
      </QueryState>
    </Page>
  );
}
