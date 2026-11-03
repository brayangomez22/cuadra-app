import type { ReactNode } from 'react';
import { useLocations } from '@/features/inventory';
import { formatDay, formatMoney } from '@/shared/format';
import { Badge } from '@/shared/ui';
import { useSupplier } from '../../../../api/suppliers';
import type { Purchase } from '../../../../api/types';
import { paymentLabels, statusLabels } from '../../../../model/purchaseLabels';
import { statusTone } from '../../../../model/purchaseStatus/purchaseStatus';
import styles from './InvoiceSummary.module.scss';

/**
 * "Datos de la factura": who sold, where the goods went, the days, how it is
 * paid and, on credit, what is left to pay: paying needs it before the lines.
 */
export function InvoiceSummary({ purchase }: { purchase: Purchase }) {
  const { supplier, location } = useParties(purchase);
  return (
    <section className={styles.root} aria-label="Datos de la factura">
      <dl className={styles.data}>
        <Datum term="Proveedor" wide>
          {supplier}
        </Datum>
        <Datum term="Sede">{location}</Datum>
        <Datum term="Fecha de la factura">{formatDay(purchase.issued_on)}</Datum>
        <Datum term="Forma de pago">{paymentLabels[purchase.payment]}</Datum>
        {purchase.due_on && <Datum term="Vence">{formatDay(purchase.due_on)}</Datum>}
        {purchase.payment === 'credit' && purchase.status !== 'voided' && (
          <Datum term="Saldo" strong>
            {formatMoney(purchase.balance_due)}
          </Datum>
        )}
        <Datum term="Estado">
          <Badge tone={statusTone(purchase.status)}>{statusLabels[purchase.status]}</Badge>
        </Datum>
        {purchase.notes && <Datum term="Notas">{purchase.notes}</Datum>}
      </dl>
    </section>
  );
}

// The names of the supplier and the location, which the purchase has only by id; "…" while they load.
function useParties(purchase: Purchase) {
  const supplier = useSupplier(purchase.supplier_id);
  const locations = useLocations();
  const location = locations.data?.find((l) => l.id === purchase.location_id);
  return { supplier: supplier.data?.name ?? '…', location: location?.name ?? '…' };
}

type DatumProps = { term: string; children: ReactNode; wide?: boolean; strong?: boolean };

// wide: a long name takes two columns instead of wrapping word by word.
function Datum({ term, children, wide, strong }: DatumProps) {
  return (
    <div className={styles.datum} data-wide={wide ? true : undefined} data-strong={strong ? true : undefined}>
      <dt>{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}
