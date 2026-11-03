import { z } from 'zod';

/** Why the purchase is voided. */
export const voidPurchaseSchema = z.object({
  reason: z.string().trim().min(1, 'Escriba el motivo').max(500, 'Use máximo 500 caracteres'),
});
export type VoidPurchaseValues = z.output<typeof voidPurchaseSchema>;
