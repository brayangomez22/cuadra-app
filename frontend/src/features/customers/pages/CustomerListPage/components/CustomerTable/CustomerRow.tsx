import { paths } from '@/app/routes/paths';
import { Badge, ButtonLink, TableCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { Customer } from '../../../../api/types';
import { documentLabel } from '../../../../model/documentLabel';
import { phoneLabel } from '../../../../model/phoneLabel';
import styles from './CustomerTable.module.scss';

type Props = {
  customer: Customer;
  segment: string | undefined;
  canWrite: boolean;
  /** Whether the user sees customers' credit (receivables:read): each row links to its account. */
  canReadCredit: boolean;
};

/**
 * A customer of the list: the name with its trade name below. For who edits
 * customers, the name is the link to its form. On a phone the document, the
 * segment and the phone go under the name, next to the way to its account.
 */
export function CustomerRow({ customer, segment, canWrite, canReadCredit }: Props) {
  const phone = customer.phone && phoneLabel(customer.phone);
  const contact = [phone, customer.email].filter(Boolean);
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
        <TableNarrow>{[documentLabel(customer), segment, phone].filter(Boolean).join(' · ')}</TableNarrow>
      </TableCell>
      <TableCell wide>
        <span className={styles.document}>{documentLabel(customer)}</span>
      </TableCell>
      <TableCell wide>{segment ?? '—'}</TableCell>
      <TableCell wide>
        {contact.length === 0
          ? '—'
          : contact.map((value) => (
              <span key={value} className={styles.contact}>
                {value}
              </span>
            ))}
      </TableCell>
      {canReadCredit && (
        <TableCell className={styles.account}>
          <ButtonLink
            variant="ghost"
            to={paths.receivables.account(customer.id)}
            aria-label={`Cuenta de ${customer.name}`}
          >
            Cuenta
          </ButtonLink>
        </TableCell>
      )}
    </TableRow>
  );
}
