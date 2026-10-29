/** A supplier's term as the lists show it: "Contado", "1 día", "30 días". */
export function paymentTermLabel(days: number): string {
  if (days === 0) return 'Contado';
  return days === 1 ? '1 día' : `${String(days)} días`;
}
