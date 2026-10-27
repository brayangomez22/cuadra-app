import { useStock } from '../../../api/stock';
import { useRowProducts } from '../../../hooks/useRowProducts';

/** A page of a location's stock, with the products its rows belong to. */
export function useStockRows(locationId: string, offset: number) {
  const stock = useStock(locationId, offset);
  const levels = stock.data?.items ?? [];
  const { products, productsById } = useRowProducts(levels, stock.isPlaceholderData);
  return { stock, levels, products, productsById };
}
