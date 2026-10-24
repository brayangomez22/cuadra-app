import { useProducts } from '@/features/catalog';
import { useStock } from '../../../api/stock';
import { PAGE_SIZE } from '../../../api/types';

/** A page of a location's stock, with the products its rows belong to. */
export function useStockRows(locationId: string, offset: number) {
  const stock = useStock(locationId, offset);
  const levels = stock.data?.items ?? [];
  // The stock has only ids: the names of its products come in one request for
  // the whole page, each product once however many of its variants it shows.
  const productIds = [...new Set(levels.map((level) => level.product_id))];
  const products = useProducts(
    { ids: productIds, status: 'all', limit: PAGE_SIZE },
    { enabled: productIds.length > 0 && !stock.isPlaceholderData },
  );
  const productsById = new Map((products.data?.items ?? []).map((product) => [product.id, product]));

  return { stock, levels, products, productsById };
}
