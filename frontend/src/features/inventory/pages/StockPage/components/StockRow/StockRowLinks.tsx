import { paths } from '@/app/routes/paths';
import { Inline, TextLink } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';

type Props = { level: StockLevel; name?: string };

/** Adjust or transfer this row's variant, with the product and the location already chosen. */
export function StockRowLinks({ level, name = 'este producto' }: Props) {
  const query = { producto: level.product_id, variante: level.variant_id, sede: level.location_id };
  return (
    <Inline gap="4" wrap>
      <TextLink tone="quiet" to={paths.inventory.adjustment(query)} aria-label={`Ajustar ${name}`}>
        Ajustar
      </TextLink>
      <TextLink tone="quiet" to={paths.inventory.transfer(query)} aria-label={`Trasladar ${name}`}>
        Trasladar
      </TextLink>
    </Inline>
  );
}
