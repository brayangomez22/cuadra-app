import { paths } from '@/app/routes/paths';
import { Badge, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { Customer } from '../../../../api/types';
import { documentLabel } from '../../../../model/documentLabel';
import styles from './CustomerTable.module.scss';

type Props = { customer: Customer; segment: string | undefined; canWrite: boolean };

/**
 * A customer of the list: the name with its trade name below. For who edits
 * customers, the name is the link to its form: on a phone it stays in view
 * while the other columns scroll.
 */
export function CustomerRow({ customer, segment, canWrite }: Props) {
  const contact = [customer.phone, customer.email].filter(Boolean);
  return (
    <TableRow>
      <TableCell>
        <span className={styles.name}>
          {canWrite ? (
            <TextLink to={paths.customers.edit(customer.id)} aria-label={`Editar ${customer.name}`}>
              {customer.name}
            </TextLink>
          ) : (
            customer.name
          )}
          {!customer.active && <Badge>Inactivo</Badge>}
        </span>
        {customer.trade_name && <span className={styles.meta}>{customer.trade_name}</span>}
      </TableCell>
      <TableCell>
        <span className={styles.document}>{documentLabel(customer)}</span>
      </TableCell>
      <TableCell>{segment ?? '—'}</TableCell>
      <TableCell>
        {contact.length === 0
          ? '—'
          : contact.map((value) => (
              <span key={value} className={styles.contact}>
                {value}
              </span>
            ))}
      </TableCell>
    </TableRow>
  );
}
