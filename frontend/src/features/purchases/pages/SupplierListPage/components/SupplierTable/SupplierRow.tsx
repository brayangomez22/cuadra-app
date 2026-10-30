import { paths } from '@/app/routes/paths';
import { formatMoney } from '@/shared/format';
import { Badge, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { Supplier } from '../../../../api/types';
import { paymentTermLabel } from '../../../../model/paymentTermLabel';
import styles from './SupplierTable.module.scss';

type Props = { supplier: Supplier; canWrite: boolean };

/** A supplier of the list. For who edits suppliers, the name is the link to its form. */
export function SupplierRow({ supplier, canWrite }: Props) {
  const contact = [supplier.contact_name, supplier.phone, supplier.email].filter(Boolean);
  return (
    <TableRow>
      <TableCell>
        <span className={styles.name}>
          {canWrite ? (
            <TextLink to={paths.purchases.editSupplier(supplier.id)} aria-label={`Editar ${supplier.name}`}>
              {supplier.name}
            </TextLink>
          ) : (
            supplier.name
          )}
          {!supplier.active && <Badge>Inactivo</Badge>}
        </span>
      </TableCell>
      <TableCell>
        <span className={styles.nit}>{supplier.nit}</span>
      </TableCell>
      <TableCell>{paymentTermLabel(supplier.payment_term_days)}</TableCell>
      <TableCell>
        {contact.length === 0
          ? '—'
          : contact.map((value) => (
              <span key={value} className={styles.contact}>
                {value}
              </span>
            ))}
      </TableCell>
      <TableCell numeric>{formatMoney(supplier.balance_due)}</TableCell>
      <TableCell>
        <TextLink
          tone="quiet"
          className={styles.purchases}
          to={paths.purchases.list({ proveedor: supplier.id })}
          aria-label={`Compras de ${supplier.name}`}
        >
          Compras
        </TextLink>
      </TableCell>
    </TableRow>
  );
}
