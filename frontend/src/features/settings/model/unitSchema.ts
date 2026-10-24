import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

export const unitSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, 'Escriba el código')
    .max(16, 'El código puede tener máximo 16 caracteres')
    .regex(/^\S+$/, 'El código va sin espacios, por ejemplo "par" o "kg"'),
  name: z.string().trim().min(1, 'Escriba el nombre').max(50, 'El nombre puede tener máximo 50 caracteres'),
});

export type UnitValues = z.infer<typeof unitSchema>;

// The field each 4xx code of /units belongs to (api/openapi.yaml).
export const unitErrorFields: FieldsByErrorCode<UnitValues> = {
  unit_code_taken: 'code',
  invalid_unit_code: 'code',
  invalid_unit_name: 'name',
};
