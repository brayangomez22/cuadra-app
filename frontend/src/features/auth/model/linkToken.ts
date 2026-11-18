/**
 * The token of an invitation or password link, from the URL's fragment
 * ("#<tenant>.<secret>"): the browser never sends the fragment, so the token
 * stays out of every server log. null when the link came incomplete.
 */
export function tokenFromHash(hash: string): string | null {
  const token = decodeURIComponent(hash.replace(/^#/, '')).trim();
  return /^[0-9a-f-]{36}\.[A-Za-z0-9_-]{43}$/.test(token) ? token : null;
}
