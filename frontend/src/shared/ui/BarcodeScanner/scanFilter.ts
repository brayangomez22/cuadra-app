/**
 * The camera sees the same code many times a second while it stays in front
 * of the lens. The filter lets each code through once, and again only after
 * it was out of view for quietMs: scanning a product twice takes moving it
 * away, as with a counter reader. With several new codes in one frame (a
 * barcode and a sticker), only the first goes through.
 */
export function createScanFilter(quietMs: number) {
  const lastSeen = new Map<string, number>();
  return (codes: string[], now: number): string | undefined => {
    const fresh = codes.find((code) => {
      const seen = lastSeen.get(code);
      return seen === undefined || now - seen > quietMs;
    });
    for (const code of codes) lastSeen.set(code, now);
    return fresh;
  };
}
