import { useAttributeDefinitions } from '@/features/catalog';
import { Field, Input, Select } from '@/shared/ui';
import { attributeChoices } from '../../../../model/attributeChoices';
import type { ProductsParams } from '../../hooks/useProductsParams';

// The value of "no attribute" in the select.
const NONE = 'ninguno';

/** A tenant attribute (marca, material, principio activo...) and the value the products must have. */
export function AttributeFilter({ params }: { params: ProductsParams }) {
  const attributes = useAttributeDefinitions();
  const chosen = attributes.data?.find((a) => a.key === params.attribute);
  const choices = chosen ? attributeChoices(chosen) : null;
  return (
    <>
      <Field label="Atributo">
        <Select
          options={[
            { value: NONE, label: 'Ninguno' },
            ...(attributes.data ?? []).map((a) => ({ value: a.key, label: a.label })),
          ]}
          value={params.attribute || NONE}
          onValueChange={(value) => params.setAttribute(value === NONE ? '' : value)}
        />
      </Field>
      {chosen && (
        <Field label={chosen.label}>
          {choices ? (
            <Select
              options={choices}
              value={params.value}
              onValueChange={params.setValue}
              placeholder="Elija un valor"
            />
          ) : (
            <Input value={params.value} onChange={(event) => params.setValue(event.target.value)} />
          )}
        </Field>
      )}
    </>
  );
}
