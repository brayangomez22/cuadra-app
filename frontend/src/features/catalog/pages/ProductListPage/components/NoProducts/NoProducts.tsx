import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconPlus, IconSearch, IconTag } from '@/shared/ui';
import type { ProductStatus } from '../../../../api/types';

type Props = { q: string; status: ProductStatus; canWrite: boolean };

/** The empty list: nothing found, no inactive products, or an empty catalog that invites to create one. */
export function NoProducts({ q, status, canWrite }: Props) {
  if (q) {
    return (
      <EmptyState
        icon={<IconSearch />}
        title={`No encontramos «${q}»`}
        description="Revise cómo lo escribió o busque por el SKU o el código de barras."
      />
    );
  }
  if (status === 'inactive') {
    return <EmptyState icon={<IconTag />} title="No hay productos inactivos" />;
  }
  return (
    <EmptyState
      icon={<IconTag />}
      title="Aún no hay productos"
      description="Cree los productos que vende para usarlos en el inventario y en las ventas."
      action={
        canWrite && (
          <ButtonLink to={paths.products.new()} icon={<IconPlus />}>
            Nuevo producto
          </ButtonLink>
        )
      }
    />
  );
}
