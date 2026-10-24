import { useParams } from 'react-router';
import { formatDateTime } from '@/shared/format';
import { QueryState } from '@/shared/query';
import { Button, IconPrinter, Page, Spinner } from '@/shared/ui';
import { CashReport } from '../../components/CashReport/CashReport';
import { Printable } from '../../components/Printable/Printable';
import styles from './CashReportPage.module.scss';
import { CashResultSummary } from './components/CashResultSummary/CashResultSummary';
import { SessionStillOpen } from './components/SessionStillOpen/SessionStillOpen';
import { useCashReport } from './hooks/useCashReport';

const title = 'Cierre de caja';

/** /caja/:sessionId: the closing report of a session, to read and to print. */
export function CashReportPage() {
  const { sessionId = '' } = useParams();
  const { session, report } = useCashReport(sessionId);

  if (!session.isSuccess) {
    return (
      <Page title={title}>
        <QueryState
          queries={[session]}
          errorTitle="No pudimos cargar el cierre"
          loading={<Spinner label="Cargando el cierre…" className={styles.loading} />}
        >
          {null}
        </QueryState>
      </Page>
    );
  }
  if (!report) {
    return (
      <Page title={title}>
        <SessionStillOpen />
      </Page>
    );
  }

  const { closing } = report.session;
  return (
    <Page
      title={title}
      description={`${report.location ? `${report.location}, cerrada` : 'Cerrada'} el ${formatDateTime(closing.closed_at)}`}
      actions={
        <Button size="lg" icon={<IconPrinter />} onClick={() => window.print()}>
          Imprimir cierre
        </Button>
      }
    >
      <CashResultSummary closing={closing} />
      <div className={styles.paper}>
        <CashReport {...report} />
      </div>
      <Printable>
        <CashReport {...report} />
      </Printable>
    </Page>
  );
}
