import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { decimalField } from '@/shared/forms/decimalField';
import { toUnits } from '@/shared/money/decimal';

// The fields hold the API's decimal only once they are valid; until then their
// own message shows and the range is not checked.
const decimal = /^\d+(\.\d{1,4})?$/;
const isDecimal = (...values: string[]) => values.every((value) => decimal.test(value));

// Same rule as ReorderRuleInput in api/openapi.yaml: 0 ≤ minimum ≤ maximum.
export const reorderRuleSchema = z
  .object({
    min_quantity: decimalField('nonnegative', {
      empty: 'Escriba el stock mínimo',
      belowMin: 'El stock mínimo no puede ser negativo',
    }),
    max_quantity: decimalField('nonnegative', {
      empty: 'Escriba el stock máximo',
      belowMin: 'El stock máximo no puede ser negativo',
    }),
  })
  .refine(
    ({ min_quantity, max_quantity }) =>
      !isDecimal(min_quantity, max_quantity) || toUnits(max_quantity) >= toUnits(min_quantity),
    {
      path: ['max_quantity'],
      message: 'El máximo no puede ser menor que el mínimo',
    },
  );

export type ReorderRuleInput = z.input<typeof reorderRuleSchema>;
export type ReorderRuleValues = z.output<typeof reorderRuleSchema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const reorderErrorFields: FieldsByErrorCode<ReorderRuleInput> = {
  invalid_reorder_range: 'min_quantity',
};
