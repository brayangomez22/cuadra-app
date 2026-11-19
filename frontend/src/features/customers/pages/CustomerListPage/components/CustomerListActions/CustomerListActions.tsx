import { paths } from '@/app/routes/paths';
import { ButtonLink, IconPlus, Inline } from '@/shared/ui';

type Props = { canWrite: boolean; canReadCredit: boolean };

/** Who owes the most (the receivables sorted by what they owe) and a new customer, for whoever may. */
export function CustomerListActions({ canWrite, canReadCredit }: Props) {
  return (
    <Inline gap="2" wrap>
      {canReadCredit && (
        <ButtonLink variant="secondary" to={paths.receivables.aging({ orden: 'total' })}>
          Quién debe más
        </ButtonLink>
      )}
      {canWrite && (
        <ButtonLink to={paths.customers.new()} icon={<IconPlus />}>
          Nuevo cliente
        </ButtonLink>
      )}
    </Inline>
  );
}
