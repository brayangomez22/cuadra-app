import { catalogVersion, eventCatalog, type EventName } from './catalog';

/**
 * The properties of event that its catalog entry allows, with the catalog's
 * version. Anything else (another key, an object, a long text) is dropped:
 * what reaches the analytics is decided here, not at each call.
 */
export function allowedProps(event: EventName, props: Record<string, unknown>) {
  const allowed: readonly string[] = eventCatalog[event];
  const out: Record<string, string | number | boolean> = { catalog_version: catalogVersion };
  for (const [key, value] of Object.entries(props)) {
    if (!allowed.includes(key)) continue;
    if (typeof value === 'number' || typeof value === 'boolean') out[key] = value;
    // Short codes and paths only: a long text could be something the user wrote.
    else if (typeof value === 'string' && value.length <= 64) out[key] = value;
  }
  return out;
}
