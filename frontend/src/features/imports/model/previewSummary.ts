import type { Import } from '../api/types';

const count = new Intl.NumberFormat('es-CO');

const plural = (n: number, one: string, many: string) => `${count.format(n)} ${n === 1 ? one : many}`;

/** What the preview says above the table: whether the file can be imported and how big it is. */
export function previewSummary({ status, counts }: Import): { title: string; text: string } {
  const size = `${plural(counts.products, 'producto', 'productos')} en ${plural(counts.rows, 'fila', 'filas')}`;
  if (status === 'validated') {
    return { title: 'El archivo está listo', text: `${size}. Revise los primeros y confirme para importarlo.` };
  }
  return {
    title: `El archivo tiene ${plural(counts.errors, 'error', 'errores')}`,
    text: `${size}. No se importó nada: corrija el archivo y súbalo de nuevo.`,
  };
}
