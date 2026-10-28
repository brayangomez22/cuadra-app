import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconPlus, IconSearch, IconUsers } from '@/shared/ui';

type Props = { q: string; filtered: boolean; canWrite: boolean };

/** The empty list: nothing found, nothing in the filter, or no customers yet, which invites to create one. */
export function NoCustomers({ q, filtered, canWrite }: Props) {
  if (q) {
    return (
      <EmptyState
        icon={<IconSearch />}
        title={`No encontramos «${q}»`}
        description="Revise cómo lo escribió o busque por el número de documento."
      />
    );
  }
  if (filtered) return <EmptyState icon={<IconUsers />} title="No hay clientes con ese filtro" />;
  return (
    <EmptyState
      icon={<IconUsers />}
      title="Todavía no hay clientes"
      description="Una venta sin cliente es a consumidor final. Guarde a quienes le piden factura o le compran seguido."
      action={
        canWrite && (
          <ButtonLink to={paths.customers.new()} icon={<IconPlus />}>
            Crear el primer cliente
          </ButtonLink>
        )
      }
    />
  );
}
