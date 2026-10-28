import { useState } from 'react';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { describeError } from '@/shared/api/errors';
import { useStartStockCount } from '../../../api/stockCounts';
import type { StockCount } from '../../../api/types';

export type CountScope = 'all' | 'categories';

/** The choices of a new count; starting it opens the counting screen. */
export function useStartCountForm(locationId: string) {
  const navigate = useNavigate();
  const start = useStartStockCount();
  const [scope, setScope] = useState<CountScope>('all');
  const [categoryIds, setCategoryIds] = useState<string[]>([]);
  const [mode, setMode] = useState<StockCount['mode']>('sum');
  const [error, setError] = useState<string>();

  const toggleCategory = (id: string, checked: boolean) =>
    setCategoryIds((ids) => (checked ? [...ids, id] : ids.filter((other) => other !== id)));

  const submit = async () => {
    if (scope === 'categories' && categoryIds.length === 0) {
      setError('Elija al menos una categoría.');
      return;
    }
    setError(undefined);
    try {
      const count = await start.mutateAsync({
        location_id: locationId,
        ...(scope === 'categories' && { category_ids: categoryIds }),
        mode,
      });
      void navigate(paths.inventory.count(count.id));
    } catch (failure) {
      setError(describeError(failure).message);
    }
  };

  return { scope, setScope, categoryIds, toggleCategory, mode, setMode, error, submit, isPending: start.isPending };
}
