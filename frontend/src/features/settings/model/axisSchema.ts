import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { parseLines } from './lines';

export const axisSchema = z.object({
  name: z.string().trim().min(1, 'Escriba el nombre').max(50, 'El nombre puede tener máximo 50 caracteres'),
  values: z.string().superRefine((text, ctx) => {
    const values = parseLines(text);
    if (values.length === 0) ctx.addIssue({ code: 'custom', message: 'Escriba al menos un valor' });
    else if (values.length > 200) ctx.addIssue({ code: 'custom', message: 'Máximo 200 valores' });
    else if (values.some((value) => value.length > 100)) {
      ctx.addIssue({ code: 'custom', message: 'Cada valor puede tener máximo 100 caracteres' });
    }
  }),
});

export type AxisValues = z.infer<typeof axisSchema>;

// The field each 4xx code of /variant-axes belongs to (api/openapi.yaml).
export const axisErrorFields: FieldsByErrorCode<AxisValues> = {
  axis_name_taken: 'name',
  invalid_axis_name: 'name',
  invalid_axis_allowed_values: 'values',
  axis_value_in_use: 'values',
};
