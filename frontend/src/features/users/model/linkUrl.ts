/**
 * The full address of a link for the user to open: the token goes in the
 * fragment (#), which the browser never sends to a server. path is
 * paths.acceptInvitation() or paths.resetPassword().
 */
export function linkUrl(path: string, token: string, origin = window.location.origin): string {
  return `${origin}${path}#${token}`;
}
