/**
 * The analytics events and the only properties each one may carry
 * (docs/eventos.md, version 2). A property outside its list is dropped
 * before sending: never names, emails, documents, prices of a customer or
 * what the user typed. Adding an event or a property is a change of the
 * catalog and of docs/eventos.md.
 */
export const eventCatalog = {
  signup_completed: ['business_template'],
  import_completed: ['products', 'errors'],
  sale_confirmed: ['lines', 'payment_methods', 'with_customer'],
  search_no_results: ['screen'],
  error_shown: ['code', 'screen'],
  feedback_sent: ['screen'],
  getting_started_step_completed: ['step'],
} as const satisfies Record<string, readonly string[]>;

/** The catalog's version, sent with every event so a change shows in the dashboards. */
export const catalogVersion = 2;

export type EventName = keyof typeof eventCatalog;

type Value = string | number | boolean;

/** The properties an event may carry, all optional. */
export type EventProps<E extends EventName> = Partial<Record<(typeof eventCatalog)[E][number], Value>>;
