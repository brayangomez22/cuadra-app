import { useFieldArray, useFormContext } from 'react-hook-form';
import { Button, FormSection, IconPlus, Stack } from '@/shared/ui';
import { emptyLine } from '../../../../model/purchaseForm/purchaseDefaults';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import { PurchaseLineRow } from '../PurchaseLineRow/PurchaseLineRow';

/** "Productos": a line per product of the invoice, in the unit it was bought in. */
export function LinesSection() {
  const { control } = useFormContext<PurchaseInputValues>();
  const { fields, append, remove } = useFieldArray({ control, name: 'lines' });
  return (
    <FormSection title="Productos" hint="El costo es por unidad de compra y sin IVA, como dice la factura.">
      <Stack gap="4">
        {fields.map((field, index) => (
          <PurchaseLineRow
            key={field.id}
            index={index}
            onRemove={fields.length > 1 ? () => remove(index) : undefined}
          />
        ))}
        <div>
          <Button type="button" variant="secondary" icon={<IconPlus />} onClick={() => append(emptyLine())}>
            Agregar línea
          </Button>
        </div>
      </Stack>
    </FormSection>
  );
}
