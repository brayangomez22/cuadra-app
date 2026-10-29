import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { isZero } from '@/shared/format';
import { ButtonLink, IconEdit } from '@/shared/ui';
import type { CustomerCredit } from '../../../../api/types';
import { PaymentDialog } from '../PaymentDialog/PaymentDialog';

type Props = { customerId: string; credit: CustomerCredit | undefined };

/** The account's actions: receive a payment while the customer owes (receivables:collect) and edit the customer. */
export function AccountActions({ customerId, credit }: Props) {
  const can = useCan();
  const owes = credit && !isZero(credit.balance);
  return (
    <>
      {owes && can('receivables:collect') && <PaymentDialog customerId={customerId} balance={credit.balance} />}
      {can('customers:write') && (
        <ButtonLink variant="secondary" icon={<IconEdit />} to={paths.customers.edit(customerId)}>
          Editar cliente
        </ButtonLink>
      )}
    </>
  );
}
