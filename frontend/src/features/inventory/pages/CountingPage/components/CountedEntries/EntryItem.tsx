import type { Product } from '@/features/catalog';
import { describeError } from '@/shared/api/errors';
import { formatDateTime, formatQuantity } from '@/shared/format';
import { IconButton, IconTrash, Skeleton, useToast } from '@/shared/ui';
import { useDeleteCountEntry } from '../../../../api/countEntries';
import type { CountEntry } from '../../../../api/types';
import { rowVariant } from '../../../../model/rowVariant';
import styles from './CountedEntries.module.scss';

type Props = { countId: string; entry: CountEntry; product?: Product; loading: boolean; canDelete: boolean };

/** One thing counted: the variant, how many (in the product's unit) and when; deletable while the count is open. */
export function EntryItem({ countId, entry, product, loading, canDelete }: Props) {
  const { toast } = useToast();
  const remove = useDeleteCountEntry(countId);
  const { name } = rowVariant(product, entry.variant_id);
  const amount = [formatQuantity(entry.quantity), product?.unit.code].filter(Boolean).join(' ');
  const shown = name ?? 'Producto';

  return (
    <li className={styles.item}>
      <span className={styles.name}>{!name && loading ? <Skeleton className={styles.skeleton} /> : shown}</span>
      <span className={styles.amount}>{amount}</span>
      <span className={styles.time}>{formatDateTime(entry.counted_at)}</span>
      {canDelete && (
        <IconButton
          label={`Borrar ${formatQuantity(entry.quantity)} de ${shown}`}
          icon={<IconTrash />}
          variant="ghost"
          className={styles.remove}
          disabled={remove.isPending}
          onClick={() =>
            remove.mutate(entry.id, {
              onError: (error) =>
                toast({ tone: 'danger', title: 'No se pudo borrar', description: describeError(error).message }),
            })
          }
        />
      )}
    </li>
  );
}
