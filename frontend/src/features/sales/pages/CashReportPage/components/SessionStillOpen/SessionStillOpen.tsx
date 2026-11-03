import { paths } from '@/app/routes/paths';
import { useAuth, useCan } from '@/features/auth';
import { ButtonLink, EmptyState, IconCash } from '@/shared/ui';
import type { CashSession } from '../../../../api/types';

/**
 * An open session has no report yet: it comes with the count, when it closes.
 * Who manages cash can close one another user left open.
 */
export function SessionStillOpen({ session }: { session: CashSession }) {
  const can = useCan();
  const auth = useAuth();
  const mine = auth.status === 'authenticated' && auth.user.id === session.user_id;
  const action =
    can('cash:manage') && !mine ? (
      <ButtonLink to={paths.sales.closeCashOf(session.id)}>Cerrar esta caja</ButtonLink>
    ) : (
      <ButtonLink to={paths.sales.cash()} variant="secondary">
        Ir a la caja
      </ButtonLink>
    );
  return (
    <EmptyState
      icon={<IconCash />}
      title="La caja sigue abierta"
      description="El reporte aparece cuando la caja se cierra con el conteo del efectivo."
      action={action}
    />
  );
}
