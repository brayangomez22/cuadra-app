import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

// The same limits as PATCH /tenant (api/openapi.yaml): empty clears a field.
export const contactSchema = z.object({
  address: z.string().trim().max(200, 'La dirección puede tener máximo 200 caracteres'),
  city: z.string().trim().max(100, 'La ciudad puede tener máximo 100 caracteres'),
  phone: z
    .string()
    .trim()
    .max(30, 'El teléfono puede tener máximo 30 caracteres')
    .regex(/^$|^[\d +()-]*\d[\d +()-]*$/, 'Escriba el teléfono solo con números, espacios, +, - o paréntesis'),
});

export type ContactValues = z.infer<typeof contactSchema>;

// The field each 422 code of PATCH /tenant belongs to.
export const contactErrorFields: FieldsByErrorCode<ContactValues> = {
  invalid_tenant_address: 'address',
  invalid_tenant_phone: 'phone',
};
