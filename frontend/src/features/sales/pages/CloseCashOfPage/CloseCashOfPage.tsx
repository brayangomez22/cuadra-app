import { useState } from 'react';
import { Navigate, useParams } from 'react-router';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useCashSession } from '../../api/cashSessions';
import { CountForm } from '../../components/CashCount/CountForm/CountForm';
import { useDrawerOwner } from '../../hooks/useDrawerOwner';
import { whereAndSinceSentence } from '../../model/cash/drawerOwner';

const notesHint = 'Por qué la cierra usted y lo que el cajero deba saber.';

/**
 * /caja/:sessionId/cerrar: who manages cash counts a drawer another user left
 * open, knowing whose it is, where and since when. A closed one goes to its report.
 */
export function CloseCashOfPage() {
  const { sessionId = '' } = useParams();
  const session = useCashSession(sessionId);
  const owner = useDrawerOwner(session.data);
  const [closed, setClosed] = useState(false);

  if (closed || session.data?.status === 'closed') return <Navigate to={paths.sales.cashReport(sessionId)} replace />;
  return (
    <Page
      title={`Cerrar la caja de ${owner?.cashier ?? 'otro usuario'}`}
      description={`${whereAndSinceSentence(owner)}Cuente el efectivo del cajón: el cierre queda a su nombre y Cuadra lo compara con lo esperado.`}
    >
      <QueryState
        queries={[session]}
        errorTitle="No pudimos cargar la caja"
        loading={<Spinner label="Cargando la caja…" />}
      >
        {() =>
          session.data && (
            <CountForm
              session={session.data}
              cashier={owner?.cashier}
              notesHint={notesHint}
              onClosed={() => setClosed(true)}
            />
          )
        }
      </QueryState>
    </Page>
  );
}
