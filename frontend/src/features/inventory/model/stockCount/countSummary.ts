import { formatPercent } from '@/shared/format';

/** "95 %": the share of the counted products that matched; a count with nothing counted has none. */
export function accuracyText(accuracy: string | undefined): string {
  return accuracy === undefined ? 'Sin contar' : formatPercent(accuracy, 'rate');
}

/** "12 de 120 productos contados": how far a count went. */
export function countedText({ lines, counted_lines }: { lines: number; counted_lines: number }): string {
  if (lines === 0) return 'Aún no hay productos en el conteo';
  const noun = lines === 1 ? 'producto contado' : 'productos contados';
  return `${String(counted_lines)} de ${String(lines)} ${noun}`;
}
