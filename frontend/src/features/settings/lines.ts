/** The non-empty lines of a text box, trimmed: the options of a list or the values of an axis. */
export function parseLines(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line !== '');
}

/** One value per line, for the text box. */
export function joinLines(values: readonly string[]): string {
  return values.join('\n');
}
