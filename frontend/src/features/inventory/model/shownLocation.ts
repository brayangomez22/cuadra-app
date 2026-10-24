import type { SelectOption } from '@/shared/ui';
import type { Location } from '../api/types';

/** The location of the URL (?sede=) or, without a valid one, the first active one. */
export function shownLocation(locations: Location[], id: string): Location | undefined {
  return locations.find((l) => l.id === id) ?? locations.find((l) => l.active) ?? locations[0];
}

/** Every location for a Select; an inactive one says so. */
export function locationOptions(locations: Location[]): SelectOption[] {
  return locations.map((l) => ({ value: l.id, label: l.active ? l.name : `${l.name} (inactiva)` }));
}
