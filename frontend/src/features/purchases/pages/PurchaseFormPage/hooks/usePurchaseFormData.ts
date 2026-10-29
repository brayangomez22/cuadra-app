import { useLocations } from '@/features/inventory';
import type { SelectOption } from '@/shared/ui';
import { useSuppliers } from '../../../api/suppliers';
import type { Supplier } from '../../../api/types';

export type PurchaseFormData = { suppliers: Supplier[]; locations: SelectOption[] };

// Only active suppliers take purchases. A business has a few dozen; more
// than 100 would need a search (docs/plan/pendientes.md, T33a).
const activeSuppliers = { status: 'active', limit: 100 } as const;

/** What the form needs: the active suppliers and locations. data waits for both. */
export function usePurchaseFormData() {
  const suppliers = useSuppliers(activeSuppliers);
  const locations = useLocations();
  const ready = suppliers.data && locations.data;
  const data: PurchaseFormData | undefined = ready
    ? {
        suppliers: suppliers.data.items,
        locations: locations.data.filter((l) => l.active).map((l) => ({ value: l.id, label: l.name })),
      }
    : undefined;
  return { queries: [suppliers, locations], data };
}
