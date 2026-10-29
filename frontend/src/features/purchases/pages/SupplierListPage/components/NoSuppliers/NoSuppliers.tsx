import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconPlus, IconSearch, IconTruck } from '@/shared/ui';

type Props = { q: string; filtered: boolean; canWrite: boolean };

/** The empty list: nothing found, nothing in the filter, or no suppliers yet, which invites to create one. */
export function NoSuppliers({ q, filtered, canWrite }: Props) {
  if (q) {
    return (
      <EmptyState
        icon={<IconSearch />}
        title={`No encontramos «${q}»`}
        description="Revise cómo lo escribió o busque por el NIT."
      />
    );
  }
  if (filtered) return <EmptyState icon={<IconTruck />} title="No hay proveedores con ese filtro" />;
  return (
    <EmptyState
      icon={<IconTruck />}
      title="Todavía no hay proveedores"
      description="Registre a quién le compra para cargar sus facturas y saber cuánto le debe."
      action={
        canWrite && (
          <ButtonLink to={paths.purchases.newSupplier()} icon={<IconPlus />}>
            Crear el primer proveedor
          </ButtonLink>
        )
      }
    />
  );
}
