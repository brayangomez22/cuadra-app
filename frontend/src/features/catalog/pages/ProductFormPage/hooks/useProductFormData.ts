import { useAuth, useModule } from '@/features/auth';
import { useAttributeDefinitions } from '../../../api/attributes';
import { useCategories } from '../../../api/categories';
import { useProduct } from '../../../api/products';
import { useUnits } from '../../../api/units';
import { useVariantAxes } from '../../../api/variantAxes';
import type { AttributeDefinition, Category, Product, UnitOfMeasure, VariantAxis } from '../../../api/types';

/** What the product form needs, once loaded. */
export type ProductFormData = {
  product?: Product;
  categories: Category[];
  units: UnitOfMeasure[];
  definitions: AttributeDefinition[];
  /** The tenant's variant axes, when the module variants is on. */
  axes?: VariantAxis[];
  /** Whether the module unit_conversions is on: the variants' alternate units are shown. */
  alternateUnits: boolean;
  /** The IVA rate a new product starts with: the tenant's. */
  defaultTaxRate: string;
};

// What the form takes from the signed-in tenant: its modules and its IVA rate.
function useTenantOptions() {
  const auth = useAuth();
  return {
    variantsOn: useModule('variants'),
    alternateUnits: useModule('unit_conversions'),
    defaultTaxRate: auth.status === 'authenticated' ? auth.user.default_tax_rate : '0.19',
  };
}

// The tenant's lists once the three are loaded.
function loadedLists(categories?: Category[], units?: UnitOfMeasure[], definitions?: AttributeDefinition[]) {
  return categories && units && definitions ? { categories, units, definitions } : undefined;
}

/** Loads the product (when editing) and the tenant's lists the form offers. data is there once all are. */
export function useProductFormData(productId: string | undefined) {
  const { variantsOn, ...options } = useTenantOptions();
  const product = useProduct(productId);
  const categories = useCategories();
  const units = useUnits();
  const definitions = useAttributeDefinitions();
  const axes = useVariantAxes({ enabled: variantsOn });
  const queries = [...(productId ? [product] : []), categories, units, definitions, ...(variantsOn ? [axes] : [])];
  const lists = loadedLists(categories.data, units.data, definitions.data);
  const data: ProductFormData | undefined =
    lists && queries.every((query) => query.isSuccess)
      ? { ...lists, ...options, product: product.data, axes: variantsOn ? axes.data : undefined }
      : undefined;
  return { queries, data };
}
