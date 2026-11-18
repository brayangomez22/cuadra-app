import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { Figure, Figures, TextLink } from '@/shared/ui';
import type { Sale } from '../../../../api/types';
import { methodLabels } from '../../../../model/payments/paymentMethods';

type Props = { sale: Sale; cashier?: string; location: string };

/**
 * The sale at a glance beside its receipt (P150): who sold, where, how it was
 * paid and, for whoever reviews the drawers, the cash session it went into.
 */
export function SaleFacts({ sale, cashier, location }: Props) {
  const can = useCan();
  const methods = [...new Set(sale.payments.map((p) => methodLabels[p.method]))].join(', ');
  return (
    <Figures label="Datos de la venta">
      <Figure term="Cajero">{cashier ?? 'Usuario borrado'}</Figure>
      {location && <Figure term="Sede">{location}</Figure>}
      <Figure term="Medio de pago">{methods}</Figure>
      {sale.cash_session_id && can('cash:manage') && (
        <Figure term="Caja">
          <TextLink to={paths.sales.cashReport(sale.cash_session_id)}>Ver la caja</TextLink>
        </Figure>
      )}
    </Figures>
  );
}
