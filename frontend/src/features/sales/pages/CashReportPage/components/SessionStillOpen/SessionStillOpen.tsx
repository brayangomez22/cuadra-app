import { paths } from '@/app/routes/paths';
import { useAuth, useCan } from '@/features/auth';
import { ButtonLink, EmptyState, IconCash } from '@/shared/ui';
import type { CashSession } from '../../../../api/types';
import { useDrawerOwner } from '../../../../hooks/useDrawerOwner';
import { openedBy } from '../../../../model/cash/drawerOwner';

/**
 * An open session has no report yet: it comes with the count, when it closes.
 * Who manages cash sees whose it is and can close one another user left open.
 */
export function SessionStillOpen({ session }: { session: CashSession }) {
  const auth = useAuth();
  const mine = auth.status === 'authenticated' && auth.user.id === session.user_id;
  const closesIt = useCan()('cash:manage') && !mine;
  const owner = useDrawerOwner(session, { enabled: closesIt });
  return (
    <EmptyState
      icon={<IconCash />}
      title="La caja sigue abierta"
      description={`${openedBy(owner)}El reporte aparece cuando la caja se cierra con el conteo del efectivo.`}
      action={closesIt ? <CloseIt session={session} cashier={owner?.cashier} /> : <GoToCash />}
    />
  );
}

function CloseIt({ session, cashier }: { session: CashSession; cashier: string | undefined }) {
  return (
    <ButtonLink to={paths.sales.closeCashOf(session.id)}>
      {cashier ? `Cerrar la caja de ${cashier}` : 'Cerrar esta caja'}
    </ButtonLink>
  );
}

function GoToCash() {
  return (
    <ButtonLink to={paths.sales.cash()} variant="secondary">
      Ir a la caja
    </ButtonLink>
  );
}
