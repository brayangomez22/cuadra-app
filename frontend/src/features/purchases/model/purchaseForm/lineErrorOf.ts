import { ApiError } from '@/shared/api/errors';

/**
 * The line (from 0) an error of POST /purchases belongs to: the API puts it
 * in details.line as text ("2", docs/decisiones.md, T33). Undefined for any other error.
 */
export function lineErrorOf(error: unknown): number | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const line = error.details.line;
  return line !== undefined && /^\d+$/.test(line) ? Number(line) : undefined;
}
