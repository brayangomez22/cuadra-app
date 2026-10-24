import { useSearchParams } from 'react-router';
import { useProduct, type Product, type Sellable } from '@/features/catalog';
import { useLocations } from '../../../api/locations';
import type { Location } from '../../../api/types';
import { shownVariant } from '../../../model/shownVariant';

/**
 * What the form starts with: the active locations and, coming from the
 * kardex or the stock, the product (?producto=, with ?variante=) and the
 * location (?sede=). With a single location it is chosen already.
 */
export function useReceiptSetup() {
  const [params] = useSearchParams();
  const locations = useLocations();
  const preset = useProduct(params.get('producto') ?? undefined);
  const active = (locations.data ?? []).filter((location) => location.active);

  return {
    locations,
    presetPending: params.has('producto') && preset.isPending,
    options: active.map((location) => ({ value: location.id, label: location.name })),
    initialItem: preset.data ? presetItem(preset.data, params.get('variante')) : null,
    initialLocation: presetLocation(active, params.get('sede')),
  };
}

function presetItem(product: Product, variantId: string | null): Sellable {
  return { product, variant: shownVariant(product, variantId) };
}

// The location of the URL if it is active; else the only one, if there is a single one.
function presetLocation(active: Location[], id: string | null): string {
  const chosen = active.find((location) => location.id === id) ?? (active.length === 1 ? active[0] : undefined);
  return chosen?.id ?? '';
}
