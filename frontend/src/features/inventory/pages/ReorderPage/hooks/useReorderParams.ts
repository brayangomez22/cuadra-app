import { enumParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import type { Location, ReorderSort } from '../../../api/types';
import { useStockLocation } from '../../../hooks/useStockLocation';

/** The value of ?sede= that lists every location at once. */
export const ALL_LOCATIONS = 'todas';

// ?orden=pedir puts first what is most to order; the default, the most urgent.
const orderParam = enumParam(['urgencia', 'pedir'], 'urgencia');
const resetPage = { reset: ['pagina'] };

/**
 * The location (?sede=, a location or every one with "todas"), the order
 * (?orden=) and the page of the reorder list, kept in the URL so the home
 * screen can link to every location.
 */
export function useReorderParams(locations: Location[]) {
  const stock = useStockLocation(locations);
  const [locationId] = useSearchParamsState('sede', stringParam(), resetPage);
  const [order, setOrder] = useSearchParamsState('orden', orderParam, resetPage);
  const all = locationId === ALL_LOCATIONS;
  const sort: ReorderSort = order === 'pedir' ? 'to_order' : 'urgency';
  return {
    ...stock,
    /** The location shown; undefined with every location. */
    location: all ? undefined : stock.location,
    all,
    order,
    sort,
    chooseOrder: (value: string) => {
      setOrder(orderParam.parse(value) ?? 'urgencia');
    },
  };
}
