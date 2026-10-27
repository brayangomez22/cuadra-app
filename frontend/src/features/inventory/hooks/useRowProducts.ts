import { useProducts } from '@/features/catalog';
import { PAGE_SIZE } from '../api/types';

/**
 * The products of a page of rows that only have ids: their names come in one
 * request for the whole page, each product once however many of its variants
 * it shows. While the page is a placeholder (another page loading), it waits.
 */
export function useRowProducts(rows: { product_id: string }[], isPlaceholder: boolean) {
  const productIds = [...new Set(rows.map((row) => row.product_id))];
  const products = useProducts(
    { ids: productIds, status: 'all', limit: PAGE_SIZE },
    { enabled: productIds.length > 0 && !isPlaceholder },
  );
  const productsById = new Map((products.data?.items ?? []).map((product) => [product.id, product]));
  return { products, productsById };
}
