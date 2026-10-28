import { Button } from '@/shared/ui';
import type { Product } from '../../../../api/types';
import { useProductStatus } from '../../hooks/useProductStatus';

/** Activates an inactive product or deactivates an active one. */
export function ProductStatusButton({ product }: { product: Product }) {
  const status = useProductStatus(product);
  return (
    <Button variant="secondary" loading={status.isPending} onClick={status.toggle}>
      {product.active ? 'Desactivar' : 'Activar'}
    </Button>
  );
}
