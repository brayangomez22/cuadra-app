import type { StockMovement } from './api';

// How each movement type reads in the kardex.
export const movementLabels: Record<StockMovement['type'], string> = {
  purchase_in: 'Entrada de mercancía',
  sale_out: 'Venta',
  adjustment: 'Ajuste',
  transfer_out: 'Traslado (sale)',
  transfer_in: 'Traslado (entra)',
};
