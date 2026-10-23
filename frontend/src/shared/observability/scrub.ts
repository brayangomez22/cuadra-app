import type { Breadcrumb, ErrorEvent } from '@sentry/react';

// The same rules as the backend's filter (backend/internal/platform/errreport/scrub.go):
// keep them in sync.

const FILTERED = '[filtered]';

// Order matters: the email rule would otherwise eat "user:password@host" partially.
const TEXT_RULES: [RegExp, string][] = [
  [/:\/\/[^/\s:@]+:[^/\s@]+@/g, `://${FILTERED}@`],
  [/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g, '[email]'],
  [/\bbearer\s+[A-Za-z0-9._~+/=-]+/gi, 'Bearer [token]'],
  [/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*/g, '[token]'],
  // Refresh token: <tenant uuid>.<32 random bytes in base64url>.
  [/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\.[A-Za-z0-9_-]{32,}/g, '[token]'],
  [/\b(password|passwd|pwd|secret|token|api[_-]?key)(\s*[=:]\s*)\S+/gi, `$1$2${FILTERED}`],
];

// Substrings of keys whose values are never sent.
const SENSITIVE_KEYS = [
  'password',
  'passwd',
  'secret',
  'token',
  'authorization',
  'cookie',
  'dsn',
  'apikey',
  'api_key',
  'email',
];

// The only request headers kept; the rest may carry credentials or personal data.
const ALLOWED_HEADERS = new Set(['User-Agent', 'Content-Type', 'Accept']);

function scrubText(text: string): string {
  return TEXT_RULES.reduce((s, [re, repl]) => s.replace(re, repl), text);
}

function isSensitiveKey(key: string): boolean {
  const k = key.toLowerCase();
  return SENSITIVE_KEYS.some((s) => k.includes(s));
}

// stripQuery drops the query string and fragment of a URL or path; other strings
// are returned as they are.
function stripQuery(value: string): string {
  if (!value.startsWith('/') && !value.includes('://')) return value;
  return value.replace(/[?#].*$/s, '');
}

// scrubValue filters sensitive keys and scrubs strings, arrays and nested objects.
function scrubValue(value: unknown): unknown {
  if (typeof value === 'string') return scrubText(stripQuery(value));
  if (Array.isArray(value)) return value.map(scrubValue);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, isSensitiveKey(k) ? FILTERED : scrubValue(v)]));
  }
  return value;
}

function scrubRecord<T extends Record<string, unknown>>(record: T | undefined): T | undefined {
  return record && (scrubValue(record) as T);
}

// scrubEvent is the beforeSend hook: it removes personal data and secrets from an
// event before it leaves the browser. It keeps only the user's opaque ID, and of
// the request only its method, path and harmless headers.
export function scrubEvent(event: ErrorEvent): ErrorEvent {
  if (event.message) event.message = scrubText(event.message);
  for (const exception of event.exception?.values ?? []) {
    if (exception.value) exception.value = scrubText(exception.value);
  }
  event.tags = scrubRecord(event.tags);
  event.extra = scrubRecord(event.extra);
  if (event.contexts) event.contexts = scrubValue(event.contexts) as ErrorEvent['contexts'];
  event.breadcrumbs = event.breadcrumbs?.map((b) => scrubBreadcrumb(b) ?? b);
  if (event.user) event.user = event.user.id === undefined ? {} : { id: event.user.id };

  const request = event.request;
  if (request) {
    event.request = {
      ...(request.url && { url: stripQuery(request.url) }),
      ...(request.method && { method: request.method }),
      ...(request.headers && {
        headers: Object.fromEntries(Object.entries(request.headers).filter(([k]) => ALLOWED_HEADERS.has(k))),
      }),
    };
  }
  return event;
}

// scrubBreadcrumb is the beforeBreadcrumb hook: fetch and navigation breadcrumbs
// lose their query strings, console messages their emails and tokens.
export function scrubBreadcrumb(breadcrumb: Breadcrumb): Breadcrumb | null {
  if (breadcrumb.message) breadcrumb.message = scrubText(breadcrumb.message);
  breadcrumb.data = scrubRecord(breadcrumb.data);
  return breadcrumb;
}
