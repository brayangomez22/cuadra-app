import { Button } from '@/shared/ui';
import type { Product } from '../../../../api/types';
import { useProductStatus } from '../../hooks/useProductStatus';

type Props = {
  product: Product;
  /** Called with the product as the API answered it. */
  onChange: (product: Product) => void;
};

/** Activates an inactive product or deactivates an active one. */
export function ProductStatusButton({ product, onChange }: Props) {
  const status = useProductStatus({ current: product, onChange });
  return (
    <Button variant="secondary" loading={status.isPending} onClick={status.toggle}>
      {product.active ? 'Desactivar' : 'Activar'}
    </Button>
  );
}
