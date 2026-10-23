import type { Product } from '../../../../api/types';
import { hasAxes } from '../../../../model/variants';
import type { ProductFormData } from '../../hooks/useProductFormData';
import { AttributesSection } from '../AttributesSection/AttributesSection';
import { GeneralSection } from '../GeneralSection/GeneralSection';
import { PriceSection } from '../PriceSection/PriceSection';
import { UnitsSection } from '../UnitsSection/UnitsSection';
import { VariantsSection } from '../VariantsSection/VariantsSection';

type Props = {
  data: ProductFormData;
  /** The saved product, as the API last answered it. */
  current?: Product;
  matrix: boolean;
};

/** The sections of the product form; the optional ones by the tenant's attributes and modules. */
export function ProductSections({ data, current, matrix }: Props) {
  const inactive = current?.active === false;
  // With the module variants on, the editor; without it, a product that already has them shows them.
  const showVariants = data.axes !== undefined || (current !== undefined && hasAxes(current));
  return (
    <>
      <GeneralSection categories={data.categories} units={data.units} matrix={matrix} />
      {data.definitions.length > 0 && <AttributesSection definitions={data.definitions} />}
      <PriceSection matrix={matrix} inactive={inactive} />
      {showVariants && <VariantsSection axes={data.axes} product={current} inactive={inactive} />}
      {data.alternateUnits && <UnitsSection product={current} units={data.units} />}
    </>
  );
}
