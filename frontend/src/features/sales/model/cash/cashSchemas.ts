import { z } from 'zod';
import { moneyField } from './moneyField';

/** Opening the drawer: where, and the cash it starts with. */
export const openCashSchema = z.object({
  location_id: z.string().min(1, 'Elija la sede de la caja'),
  opening_float: moneyField('nonnegative', 'Escriba la base, aunque sea 0'),
});
export type OpenCashInput = z.input<typeof openCashSchema>;
export type OpenCashValues = z.output<typeof openCashSchema>;

/** A cash in or out: how much, and why. */
export const cashMovementSchema = z.object({
  amount: moneyField('positive', 'Escriba el valor'),
  reason: z.string().trim().min(1, 'Escriba el motivo').max(200, 'Use máximo 200 caracteres'),
});
export type CashMovementFormInput = z.input<typeof cashMovementSchema>;
export type CashMovementValues = z.output<typeof cashMovementSchema>;
