import { paths } from '@/app/routes/paths';
import { ButtonLink, IconPlus, Inline } from '@/shared/ui';

/** What is owed, the suppliers and, for who registers purchases, a new one. */
export function PurchaseListActions({ canWrite }: { canWrite: boolean }) {
  return (
    <Inline wrap>
      <ButtonLink to={paths.purchases.payables()} variant="secondary">
        Cuentas por pagar
      </ButtonLink>
      <ButtonLink to={paths.purchases.suppliers()} variant="secondary">
        Proveedores
      </ButtonLink>
      {canWrite && (
        <ButtonLink to={paths.purchases.new()} icon={<IconPlus />}>
          Registrar compra
        </ButtonLink>
      )}
    </Inline>
  );
}
