import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { parseLines } from './lines';

export const attributeSchema = z
  .object({
    label: z.string().trim().min(1, 'Escriba el nombre').max(100, 'El nombre puede tener máximo 100 caracteres'),
    key: z
      .string()
      .trim()
      .min(1, 'Escriba la clave')
      .max(50, 'La clave puede tener máximo 50 caracteres')
      .regex(/^[a-z][a-z0-9_]*$/, 'Use minúsculas sin tildes, números y _, empezando por una letra: principio_activo'),
    type: z.enum(['text', 'number', 'list', 'boolean']),
    options: z.string(),
    required: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.type !== 'list') return;
    const options = parseLines(values.options);
    if (options.length === 0)
      ctx.addIssue({ code: 'custom', path: ['options'], message: 'Escriba al menos una opción' });
    else if (options.length > 100) ctx.addIssue({ code: 'custom', path: ['options'], message: 'Máximo 100 opciones' });
    else if (options.some((option) => option.length > 100)) {
      ctx.addIssue({ code: 'custom', path: ['options'], message: 'Cada opción puede tener máximo 100 caracteres' });
    }
  });

export type AttributeValues = z.infer<typeof attributeSchema>;

// The field each 4xx code of /product-attributes belongs to (api/openapi.yaml).
export const attributeErrorFields: FieldsByErrorCode<AttributeValues> = {
  attribute_key_taken: 'key',
  invalid_attribute_key: 'key',
  invalid_attribute_label: 'label',
  invalid_attribute_options: 'options',
  attribute_option_in_use: 'options',
};
