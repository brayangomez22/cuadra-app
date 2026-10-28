import { useSetProductStatus } from '../../../api/products';
import type { Product } from '../../../api/types';

/** Activates an inactive product or deactivates an active one; the product in the cache shows the change. */
export function useProductStatus(current: Product) {
  const status = useSetProductStatus();
  function toggle() {
    status.mutate({ id: current.id, active: !current.active });
  }
  return { toggle, isPending: status.isPending };
}
