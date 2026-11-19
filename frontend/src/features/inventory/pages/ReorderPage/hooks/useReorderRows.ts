import type { ReorderSort } from '../../../api/types';
import { useReorderAlerts } from '../../../api/reorder';
import { useRowProducts } from '../../../hooks/useRowProducts';

type Query = { locationId: string | undefined; offset: number; sort: ReorderSort };

/** A page of the reorder alerts (of a location, or of all), with the products its rows belong to. */
export function useReorderRows(query: Query) {
  const alerts = useReorderAlerts(query);
  const rows = alerts.data?.items ?? [];
  const { products, productsById } = useRowProducts(rows, alerts.isPlaceholderData);
  return { alerts, rows, products, productsById };
}
