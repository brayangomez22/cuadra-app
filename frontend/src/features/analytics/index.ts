// The public API of analytics: the session (app/ starts it with the user of
// /me) and the events of the catalog (docs/eventos.md), which any feature
// may send: without a session they go nowhere.
export type { EventName, EventProps } from './model/catalog';
export { useAnalyticsSession } from './hooks/useAnalyticsSession';
export { track, trackOnStart } from './tracker/tracker';
