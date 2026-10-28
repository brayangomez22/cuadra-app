import type { StockMovement } from '../api/types';

// How each movement type reads in the kardex.
export const movementLabels: Record<StockMovement['type'], string> = {
  purchase_in: 'Entrada de mercancía',
  initial_in: 'Inventario inicial',
  sale_out: 'Venta',
  sale_void: 'Venta anulada',
  adjustment: 'Ajuste',
  transfer_out: 'Traslado (sale)',
  transfer_in: 'Traslado (entra)',
};
