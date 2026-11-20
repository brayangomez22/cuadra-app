import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconPlus, IconSearch, IconTag, IconUpload, Inline } from '@/shared/ui';
import type { ProductStatus } from '../../../../api/types';

type Props = { q: string; status: ProductStatus; canWrite: boolean; canImport: boolean };

/** The empty list: nothing found, no inactive products, or an empty catalog that invites to import or create them. */
export function NoProducts({ q, status, canWrite, canImport }: Props) {
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
      description="Impórtelos desde Excel o créelos uno por uno para usarlos en el inventario y en las ventas."
      action={
        (canWrite || canImport) && (
          <Inline gap="2" wrap>
            {canImport && (
              <ButtonLink to={paths.imports.new()} icon={<IconUpload />}>
                Importar desde Excel
              </ButtonLink>
            )}
            {canWrite && (
              <ButtonLink to={paths.products.new()} variant={canImport ? 'secondary' : 'primary'} icon={<IconPlus />}>
                Nuevo producto
              </ButtonLink>
            )}
          </Inline>
        )
      }
    />
  );
}
