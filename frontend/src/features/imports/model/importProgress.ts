import type { ImportCounts, ImportStatus } from '../api/types';

/** What the status screen shows: the progress, the report, the failure or a link back to the upload. */
export type ImportStage = 'running' | 'completed' | 'failed' | 'unconfirmed';

export function importStage(status: ImportStatus): ImportStage {
  switch (status) {
    case 'queued':
    case 'running':
      return 'running';
    case 'completed':
    case 'failed':
      return status;
    default:
      return 'unconfirmed';
  }
}

/** The products the job is done with: the counts grow batch by batch. */
export function processed(counts: ImportCounts): number {
  return counts.created + counts.updated + counts.skipped;
}

const count = new Intl.NumberFormat('es-CO');

const lines: [keyof ImportCounts, string, string][] = [
  ['created', 'creado', 'creados'],
  ['updated', 'actualizado', 'actualizados'],
  ['skipped', 'omitido', 'omitidos'],
];

/** The final report, one line per outcome that happened: "2 creados", "1 actualizado". */
export function resultLines(counts: ImportCounts): string[] {
  return lines
    .filter(([key]) => counts[key] > 0)
    .map(([key, one, many]) => `${count.format(counts[key])} ${counts[key] === 1 ? one : many}`);
}
