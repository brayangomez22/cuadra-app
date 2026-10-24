import { defaultVariant, type Product, type ProductVariant } from '@/features/catalog';

/** The variant of the URL (?variante=) or, without a valid one, the product's default variant. */
export function shownVariant(product: Product, id: string | null | undefined): ProductVariant {
  return product.variants.find((v) => v.id === id) ?? defaultVariant(product);
}
