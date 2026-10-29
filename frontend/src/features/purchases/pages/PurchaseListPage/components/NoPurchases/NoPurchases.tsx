import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconPlus, IconTruck } from '@/shared/ui';

type Props = { filtered: boolean; canWrite: boolean };

/** The empty list: nothing in the filter, or no purchases yet, which invites to register the first one. */
export function NoPurchases({ filtered, canWrite }: Props) {
  if (filtered) return <EmptyState icon={<IconTruck />} title="No hay compras con ese filtro" />;
  return (
    <EmptyState
      icon={<IconTruck />}
      title="Todavía no hay compras"
      description="Registre la factura de un proveedor: la mercancía entra al inventario y actualiza el costo."
      action={
        canWrite && (
          <ButtonLink to={paths.purchases.new()} icon={<IconPlus />}>
            Registrar la primera compra
          </ButtonLink>
        )
      }
    />
  );
}
