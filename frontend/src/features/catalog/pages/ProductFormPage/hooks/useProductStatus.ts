import { useSetProductStatus } from '../../../api/products';
import type { Product } from '../../../api/types';

type Input = {
  current?: Product;
  /** Called with the product as the API answered it. */
  onChange: (product: Product) => void;
};

/** Activates an inactive product or deactivates an active one. */
export function useProductStatus({ current, onChange }: Input) {
  const status = useSetProductStatus();
  function toggle() {
    if (!current) return;
    status.mutate({ id: current.id, active: !current.active }, { onSuccess: onChange });
  }
  return { toggle, isPending: status.isPending };
}
