import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { TextLink } from '@/shared/ui';
import type { StockMovement } from '../../../../api/types';
import { movementLabels } from '../../../../model/movementLabels';

/** What the movement was; a purchase's entry links to its invoice, for who can see purchases. */
export function MovementLabel({ movement }: { movement: StockMovement }) {
  const can = useCan();
  const label = movementLabels[movement.type];
  if (movement.reference?.kind !== 'purchase' || !can('purchases:read')) return label;
  return (
    <TextLink tone="quiet" to={paths.purchases.detail(movement.reference.id)}>
      {label}
    </TextLink>
  );
}
