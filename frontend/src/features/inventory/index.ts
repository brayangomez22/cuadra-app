// The public API of inventory: the locations, which the point of sale and
// the cash screens use, the keys a sale invalidates and the reorder summary
// of the home screen.
export { inventoryKeys } from './api/keys';
export { useLocations } from './api/locations';
export type { Location } from './api/types';

/** How many products reached their minimum stock in every location, with a link to the list, for the home screen. */
export { ReorderSummary } from './components/ReorderSummary/ReorderSummary';
