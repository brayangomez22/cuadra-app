import { paths } from '@/app/routes/paths';
import { NoLocationsYet } from '@/features/inventory';
import { ButtonLink, EmptyState, IconCash } from '@/shared/ui';

type Props = { reason: 'cash_required' | 'no_locations' };

/** Why the POS cannot sell yet, and what to do. */
export function PosUnavailable({ reason }: Props) {
  if (reason === 'cash_required') {
    return (
      <EmptyState
        icon={<IconCash />}
        title="Abra la caja para vender"
        description="Cuente el efectivo del cajón y abra la caja. La venta en curso no se pierde."
        action={<ButtonLink to={paths.sales.cash()}>Abrir caja</ButtonLink>}
      />
    );
  }
  return (
    <NoLocationsYet title="No hay sedes activas para vender" reason="Las ventas descuentan el stock de una sede." />
  );
}
