import type { ReactNode } from 'react';
import { useLocations } from '@/features/inventory';
import { formatDay } from '@/shared/format';
import { Badge } from '@/shared/ui';
import { useSupplier } from '../../../../api/suppliers';
import type { Purchase } from '../../../../api/types';
import { paymentLabels, statusLabels } from '../../../../model/purchaseLabels';
import styles from './InvoiceSummary.module.scss';

/** "Datos de la factura": who sold, where the goods went, the days and how it is paid. */
export function InvoiceSummary({ purchase }: { purchase: Purchase }) {
  const { supplier, location } = useParties(purchase);
  const open = purchase.status === 'open';
  return (
    <section className={styles.root} aria-label="Datos de la factura">
      <dl className={styles.data}>
        <Datum term="Proveedor">{supplier}</Datum>
        <Datum term="Sede">{location}</Datum>
        <Datum term="Fecha de la factura">{formatDay(purchase.issued_on)}</Datum>
        <Datum term="Forma de pago">{paymentLabels[purchase.payment]}</Datum>
        {purchase.due_on && <Datum term="Vence">{formatDay(purchase.due_on)}</Datum>}
        <Datum term="Estado">
          <Badge tone={open ? 'warning' : 'success'}>{statusLabels[purchase.status]}</Badge>
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

function Datum({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className={styles.datum}>
      <dt>{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}
