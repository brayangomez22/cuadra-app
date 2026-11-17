/**
 * A URL without its query and fragment: they may carry what the user
 * searched (a customer's name, a document). The path only has IDs.
 */
export function cleanUrl(url: string) {
  const end = url.search(/[?#]/);
  return end === -1 ? url : url.slice(0, end);
}
