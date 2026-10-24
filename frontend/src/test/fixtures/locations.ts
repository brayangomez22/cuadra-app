import { ids } from './ids';
import type { Location } from './types';

export function aLocation(overrides: Partial<Location> = {}): Location {
  return { id: ids.main, name: 'Bodega principal', active: true, created_at: '2026-10-01T15:00:00Z', ...overrides };
}
