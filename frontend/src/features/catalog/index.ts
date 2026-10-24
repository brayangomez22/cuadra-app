// The public API of catalog: its products and variants, the tenant's lists
// (units, attributes, axes) and the picker other screens sell and count with.
// The router takes the routes from routes.tsx, which loads pages.ts lazily.
export { catalogKeys } from './api/keys';
export { useAttributeDefinitions } from './api/attributes';
export { fetchProducts, useProduct, useProducts } from './api/products';
export { useUnits } from './api/units';
export { useVariantAxes } from './api/variantAxes';
export type {
  AttributeDefinition,
  Product,
  ProductQuery,
  ProductVariant,
  UnitOfMeasure,
  VariantAxis,
  VariantUnit,
} from './api/types';
export { ProductPicker } from './components/ProductPicker/ProductPicker';
export { taxRateOptions } from './model/taxRates';
export {
  defaultVariant,
  findByCode,
  hasAxes,
  sellables,
  suggestedUnitPrice,
  variantName,
  type Sellable,
} from './model/variants';
