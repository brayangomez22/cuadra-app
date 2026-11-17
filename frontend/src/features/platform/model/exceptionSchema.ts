import { z } from 'zod';

/** The form that gives one business its own value of a flag. */
export const exceptionSchema = z.object({
  tenant_id: z
    .string()
    .trim()
    .pipe(z.uuid({ error: 'Escriba el ID de la empresa (un UUID, como aparece en su sesión).' })),
  value: z.enum(['on', 'off']),
});

export type ExceptionValues = z.infer<typeof exceptionSchema>;

export const exceptionDefaults: ExceptionValues = { tenant_id: '', value: 'on' };
