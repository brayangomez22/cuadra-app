import { paths } from '@/app/routes/paths';
import type { Product } from '@/features/catalog';
import { formatMoney, formatQuantity } from '@/shared/format';
import { Badge, TableCell, TableRow } from '@/shared/ui';
import type { StockCountLine } from '../../../../api/types';
import { VariantLabel } from '../../../../components/VariantLabel/VariantLabel';
import { lineStatusLabels } from '../../../../model/stockCount/statusLabels';
import { RecountButton } from './RecountButton';
import styles from './LinesTable.module.scss';

type Props = { countId: string; line: StockCountLine; product?: Product; loading: boolean; canRecount: boolean };

/** A product of the count: expected, counted and the difference in quantity and at cost. */
export function LineRow({ countId, line, product, loading, canRecount }: Props) {
  const status = lineStatusLabels[line.status];
  const unit = product?.unit.code;
  const amount = (value: string | undefined, signed = false) =>
    value === undefined ? '—' : [formatQuantity(value, { signed }), unit].filter(Boolean).join(' ');
  const doubtful = line.status === 'review' || line.status === 'difference';

  return (
    <TableRow>
      <TableCell>
        <VariantLabel
          product={product}
          variantId={line.variant_id}
          to={paths.inventory.kardex(line.product_id, { variante: line.variant_id })}
          loading={loading}
        />
      </TableCell>
      <TableCell>
        <Badge tone={status.tone}>{status.label}</Badge>
      </TableCell>
      <TableCell numeric>{amount(line.expected_quantity)}</TableCell>
      <TableCell numeric>{amount(line.counted_quantity)}</TableCell>
      <TableCell numeric>
        <span className={styles.difference}>{amount(line.difference, true)}</span>
      </TableCell>
      <TableCell numeric>{line.difference_value === undefined ? '—' : formatMoney(line.difference_value)}</TableCell>
      {canRecount && (
        <TableCell>
          {doubtful && <RecountButton countId={countId} variantId={line.variant_id} product={product} />}
        </TableCell>
      )}
    </TableRow>
  );
}
