import { useProducts, type Product } from '@/features/catalog';
import { useStock } from '../../../api/stock';
import { useRowProducts } from '../../../hooks/useRowProducts';

// A search asks the catalog for these many products, and the stock of at
// most these many of their variants (the API's limit).
const SEARCHED_PRODUCTS = 50;
const SEARCHED_VARIANTS = 100;

// The variants whose stock a search asks for: none while the catalog has not
// answered, and no filter (the whole location) without a term.
function variantsOf(term: string, products: Product[] | undefined): string[] | undefined {
  if (term === '') return undefined;
  return (products ?? []).flatMap((p) => p.variants.map((v) => v.id)).slice(0, SEARCHED_VARIANTS);
}

/**
 * A page of a location's stock, with the products its rows belong to. With a
 * term, the catalog finds the products (by name, SKU or barcode) and the page
 * holds only their variants; noMatch says the catalog found none.
 */
export function useStockRows(locationId: string, offset: number, term: string) {
  const found = useProducts({ q: term, status: 'all', limit: SEARCHED_PRODUCTS }, { enabled: term !== '' });
  const searched = term === '' ? undefined : found.data?.items;
  const stock = useStock(locationId, offset, variantsOf(term, searched));
  const levels = stock.data?.items ?? [];
  const { products, productsById } = useRowProducts(levels, stock.isPlaceholderData);
  return { stock, found, levels, products, productsById, noMatch: searched?.length === 0 };
}
