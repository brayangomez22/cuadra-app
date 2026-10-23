import { useParams } from 'react-router';
import { useAuth } from '@/features/auth';
import { useLocations } from '@/features/inventory';
import { describeError } from '@/shared/api/errors';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Alert, Badge, Button, ButtonLink, EmptyState, IconCash, IconPrinter, Page, Spinner } from '@/shared/ui';
import { useCashSession } from './api';
import { resultLabels, type CashResult } from './cash';
import { CashReport } from './CashReport';
import { Printable } from './Printable';
import { loadPaper } from './model/storage';
import styles from './CashReportPage.module.scss';
import { paths } from '@/app/routes/paths';

const title = 'Cierre de caja';

const resultTones: Record<CashResult, 'success' | 'danger' | 'warning'> = {
  exact: 'success',
  shortage: 'danger',
  overage: 'warning',
};

/** /caja/:sessionId: the closing report of a session, to read and to print. */
export function CashReportPage() {
  const { sessionId = '' } = useParams();
  const session = useCashSession(sessionId);
  const locations = useLocations();
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;

  if (session.isPending) {
    return (
      <Page title={title}>
        <Spinner label="Cargando el cierre…" className={styles.loading} />
      </Page>
    );
  }
  if (session.isError) {
    return (
      <Page title={title}>
        <Alert tone="danger" title="No pudimos cargar el cierre">
          {describeError(session.error).message}
        </Alert>
      </Page>
    );
  }

  const data = session.data;
  const closing = data.closing;
  if (!closing) {
    return (
      <Page title={title}>
        <EmptyState
          icon={<IconCash />}
          title="La caja sigue abierta"
          description="El reporte aparece cuando quien la abrió la cierra con el conteo del efectivo."
          action={
            <ButtonLink to={paths.sales.cash()} variant="secondary">
              Ir a la caja
            </ButtonLink>
          }
        />
      </Page>
    );
  }

  const location = locations.data?.find((l) => l.id === data.location_id)?.name ?? '';
  // Users have no endpoint yet: the cashier's name is known only for one's own drawer.
  const cashier = user && user.id === data.user_id ? user.name : undefined;
  const report = {
    session: { ...data, closing },
    business: { name: user?.tenant_name ?? '', nit: user?.tenant_nit ?? '' },
    location,
    cashier,
    paper: loadPaper(),
  };

  return (
    <Page
      title={title}
      description={`${location ? `${location}, cerrada` : 'Cerrada'} el ${formatDateTime(closing.closed_at)}`}
      actions={
        <Button size="lg" icon={<IconPrinter />} onClick={() => window.print()}>
          Imprimir cierre
        </Button>
      }
    >
      <section className={styles.result} aria-label="Resultado del arqueo">
        <Badge tone={resultTones[closing.result]}>{resultLabels[closing.result]}</Badge>
        <dl className={styles.figures}>
          <div className={styles.figure}>
            <dt>Esperado</dt>
            <dd>{formatMoney(closing.expected)}</dd>
          </div>
          <div className={styles.figure}>
            <dt>Contado</dt>
            <dd>{formatMoney(closing.counted)}</dd>
          </div>
          <div className={styles.figure} data-role="difference">
            <dt>Diferencia</dt>
            <dd>{formatMoney(closing.difference)}</dd>
          </div>
        </dl>
      </section>

      <div className={styles.paper}>
        <CashReport {...report} />
      </div>
      <Printable>
        <CashReport {...report} />
      </Printable>
    </Page>
  );
}
