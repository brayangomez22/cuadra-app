import { useState } from 'react';
import { paths } from '@/app/routes/paths';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Button, ButtonLink, Page } from '@/shared/ui';
import type { CashSession } from '../../../../api/types';
import type { CashMovementKind } from '../../../../model/cash/cashLabels';
import { CashMovementDialog } from '../CashMovementDialog/CashMovementDialog';
import { CashMovementsTable } from '../CashMovementsTable/CashMovementsTable';
import styles from './OpenSession.module.scss';

/** The open drawer: where and when it opened, its float, and its cash ins and outs. */
export function OpenSession({ session, locationName }: { session: CashSession; locationName: string }) {
  const [movement, setMovement] = useState<CashMovementKind>();

  return (
    <Page
      title="Caja"
      description="Registre aquí el efectivo que entra o sale del cajón sin ser una venta."
      actions={
        <ButtonLink to={paths.sales.closeCash()} size="lg">
          Cerrar caja
        </ButtonLink>
      }
    >
      <dl className={styles.facts}>
        <div className={styles.fact}>
          <dt>Sede</dt>
          <dd>{locationName}</dd>
        </div>
        <div className={styles.fact}>
          <dt>Abierta</dt>
          <dd>{formatDateTime(session.opened_at)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>Base inicial</dt>
          <dd className={styles.amount}>{formatMoney(session.opening_float)}</dd>
        </div>
      </dl>

      <div className={styles.movementActions}>
        <Button variant="secondary" onClick={() => setMovement('in')}>
          Registrar ingreso
        </Button>
        <Button variant="secondary" onClick={() => setMovement('out')}>
          Registrar egreso
        </Button>
      </div>

      <CashMovementsTable movements={session.movements} />

      <CashMovementDialog
        sessionId={session.id}
        kind={movement}
        onOpenChange={(open) => {
          if (!open) setMovement(undefined);
        }}
      />
    </Page>
  );
}
