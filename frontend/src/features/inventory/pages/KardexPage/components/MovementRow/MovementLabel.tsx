import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { TextLink } from '@/shared/ui';
import type { StockMovement } from '../../../../api/types';
import { movementLabels } from '../../../../model/movementLabels';

/** What the movement was; a purchase's entry links to its invoice and a sale's movement to the sale, for who can see them. */
export function MovementLabel({ movement }: { movement: StockMovement }) {
  const link = useDocumentLink(movement.reference);
  const label = movementLabels[movement.type];
  if (!link) return label;
  return (
    <TextLink tone="quiet" to={link}>
      {label}
    </TextLink>
  );
}

function useDocumentLink(reference: StockMovement['reference']) {
  const can = useCan();
  if (reference?.kind === 'purchase' && can('purchases:read')) return paths.purchases.detail(reference.id);
  if (reference?.kind === 'sale' && can('sales:read')) return paths.sales.detail(reference.id);
  return undefined;
}
