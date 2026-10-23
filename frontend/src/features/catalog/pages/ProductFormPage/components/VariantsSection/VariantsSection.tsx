import type { Product, VariantAxis } from '../../../../api/types';
import { FormSection } from '../FormSection/FormSection';
import { VariantEditor } from '../VariantEditor/VariantEditor';
import { VariantList } from '../VariantList/VariantList';

type Props = {
  /** The tenant's axes when the module variants is on: the variants are edited. */
  axes?: VariantAxis[];
  /** The saved product, whose variants show while the module is off. */
  product?: Product;
  inactive: boolean;
};

/** "Variantes": the matrix editor with the module on; without it, the variants the product already has. */
export function VariantsSection({ axes, product, inactive }: Props) {
  if (axes !== undefined) {
    return (
      <FormSection
        title="Variantes"
        hint="Si el producto se vende en varias versiones, elija sus valores: cada combinación es una variante con su propio SKU, código y precio."
      >
        <VariantEditor axes={axes} inactive={inactive} />
      </FormSection>
    );
  }
  return <FormSection title="Variantes">{product && <VariantList product={product} />}</FormSection>;
}
