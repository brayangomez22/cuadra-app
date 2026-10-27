import { useReorderAlerts } from '../../../api/reorder';
import { useRowProducts } from '../../../hooks/useRowProducts';

/** A page of a location's reorder alerts, with the products its rows belong to. */
export function useReorderRows(locationId: string, offset: number) {
  const alerts = useReorderAlerts({ locationId, offset });
  const rows = alerts.data?.items ?? [];
  const { products, productsById } = useRowProducts(rows, alerts.isPlaceholderData);
  return { alerts, rows, products, productsById };
}
