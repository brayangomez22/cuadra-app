import { formatQuantity } from '@/shared/format';
import { IconButton, IconEdit, TableCell } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';
import { ReorderRuleDialog } from '../ReorderRuleDialog/ReorderRuleDialog';
import styles from './ReorderCell.module.scss';

type Props = { level: StockLevel; name?: string; canEdit: boolean };

/** The variant's minimum and maximum at the location and, for who adjusts stock, the button to change them. */
export function ReorderCell({ level, name = 'la variante', canEdit }: Props) {
  const rule = level.reorder;
  return (
    <TableCell numeric>
      <span className={styles.root}>
        {rule ? `${formatQuantity(rule.min_quantity)} – ${formatQuantity(rule.max_quantity)}` : '—'}
        {canEdit && (
          <ReorderRuleDialog
            level={level}
            name={name}
            trigger={<IconButton label={`Mínimo y máximo de ${name}`} icon={<IconEdit />} />}
          />
        )}
      </span>
    </TableCell>
  );
}
