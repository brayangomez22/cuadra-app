import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { hasAxes, variantName, type Product } from '@/features/catalog';
import { ButtonLink, IconPlus, Page } from '@/shared/ui';
import { useKardex } from '../../../../api/kardex';
import { useLocations } from '../../../../api/locations';
import { shownVariant } from '../../../../model/shownVariant';
import { useKardexFilters } from '../../hooks/useKardexFilters';
import { KardexFilters } from '../KardexFilters/KardexFilters';
import { KardexMovements } from '../KardexMovements/KardexMovements';
import { KardexSubtitle } from '../KardexSubtitle/KardexSubtitle';

/** The kardex of a loaded product: its filters and the movements of the variant shown. */
export function ProductKardex({ product }: { product: Product }) {
  const can = useCan();
  const filters = useKardexFilters();
  const locations = useLocations();
  const variant = shownVariant(product, filters.variantId);
  const kardex = useKardex(variant.id, filters.locationId, filters.offset);
  const name = variantName(product.name, variant.label);
  const receiptLink = paths.inventory.receipt({ producto: product.id, variante: variant.id, sede: filters.locationId });

  return (
    <Page
      title={name}
      description={<KardexSubtitle product={product} variant={variant} />}
      actions={
        can('inventory:adjust') && (
          <ButtonLink to={receiptLink} icon={<IconPlus />}>
            Registrar entrada
          </ButtonLink>
        )
      }
    >
      <KardexFilters product={product} variant={variant} locations={locations} filters={filters} />
      <KardexMovements
        kardex={kardex}
        caption={`Movimientos de ${name}`}
        withAxes={hasAxes(product)}
        filters={filters}
        locations={locations.data ?? []}
        canSeeCost={can('catalog:write')}
      />
    </Page>
  );
}
