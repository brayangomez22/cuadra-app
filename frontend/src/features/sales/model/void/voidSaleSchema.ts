import { z } from 'zod';

/** Why the sale is voided. */
export const voidSaleSchema = z.object({
  reason: z.string().trim().min(1, 'Escriba el motivo').max(500, 'Use máximo 500 caracteres'),
});
export type VoidSaleValues = z.output<typeof voidSaleSchema>;
