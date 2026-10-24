// The public API of inventory: the locations, which the point of sale and
// the cash screens use, and the keys a sale invalidates.
export { inventoryKeys } from './api/keys';
export { useLocations } from './api/locations';
export type { Location } from './api/types';
