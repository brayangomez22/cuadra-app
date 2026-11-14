import { z } from 'zod';
import type { components } from '@/shared/api/schema';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

type UnitInput = components['schemas']['UnitInput'];

export const unitSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, 'Escriba el código')
    .max(16, 'El código puede tener máximo 16 caracteres')
    .regex(/^\S+$/, 'El código va sin espacios, por ejemplo "par" o "kg"'),
  name: z.string().trim().min(1, 'Escriba el nombre').max(50, 'El nombre puede tener máximo 50 caracteres'),
  // The UN/ECE Rec 20 code of the electronic invoice; empty, the invoice uses 94 (unit).
  dian_code: z
    .string()
    .trim()
    .toUpperCase()
    .regex(/^[A-Z0-9]{0,3}$/, 'El código DIAN tiene de 1 a 3 letras o números, por ejemplo MTR o KGM'),
});

export type UnitValues = z.infer<typeof unitSchema>;

// The field each 4xx code of /units belongs to (api/openapi.yaml).
export const unitErrorFields: FieldsByErrorCode<UnitValues> = {
  unit_code_taken: 'code',
  invalid_unit_code: 'code',
  invalid_unit_name: 'name',
  invalid_unit_dian_code: 'dian_code',
};

/** The body of POST and PUT /units: an empty DIAN code is left out, which leaves the unit without one. */
export function toUnitInput({ dian_code, ...rest }: UnitValues): UnitInput {
  return dian_code ? { ...rest, dian_code } : rest;
}

/** The form's values: the unit to edit, or empty to add one. */
export function unitValues(unit: { code: string; name: string; dian_code?: string } | null): UnitValues {
  return { code: unit?.code ?? '', name: unit?.name ?? '', dian_code: unit?.dian_code ?? '' };
}
