/**
 * Hands a file the API returned to the browser as a download. A link to the
 * API would not carry the session's token, so the file is asked with the
 * client first.
 */
export function saveFile(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}
