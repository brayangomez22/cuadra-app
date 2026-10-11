// Helpers for tokens.test.ts: read custom properties from compiled CSS and
// compute WCAG 2.x contrast ratios.

export type Tokens = Map<string, string>;

/**
 * Collects the custom properties declared in every rule whose selector is
 * exactly `selector`, in source order (later declarations win). Nested
 * at-rules such as @media are transparent: pass the inner selector.
 */
export function customProperties(css: string, selector: string): Tokens {
  const tokens: Tokens = new Map();
  for (const [, sel, body] of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (normalize(sel ?? '') !== normalize(selector)) continue;
    for (const [, name, value] of (body ?? '').matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
      if (name && value) tokens.set(name, value.trim());
    }
  }
  return tokens;
}

// Sass drops the quotes of attribute selectors when they are not needed.
const normalize = (selector: string) => selector.trim().replace(/['"]/g, '');

/** Returns the body of the first `@media <query>` block, or '' if there is none. */
export function mediaBlock(css: string, query: string): string {
  const start = css.indexOf(`@media ${query}`);
  if (start === -1) return '';
  let depth = 0;
  for (let i = css.indexOf('{', start); i < css.length; i++) {
    if (css[i] === '{') depth++;
    if (css[i] === '}' && --depth === 0) return css.slice(css.indexOf('{', start) + 1, i);
  }
  return '';
}

/** Removes every @media block, leaving only the rules that always apply. */
export function withoutMedia(css: string): string {
  let out = css;
  for (let start = out.indexOf('@media'); start !== -1; start = out.indexOf('@media')) {
    let depth = 0;
    let end = out.indexOf('{', start);
    for (; end < out.length; end++) {
      if (out[end] === '{') depth++;
      if (out[end] === '}' && --depth === 0) break;
    }
    out = out.slice(0, start) + out.slice(end + 1);
  }
  return out;
}

function channels(hex: string): [number, number, number] {
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex.trim());
  if (!m?.[1]) throw new Error(`"${hex}" is not a hex color`);
  const h = m[1].length === 3 ? [...m[1]].map((c) => c + c).join('') : m[1];
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16) / 255) as [number, number, number];
}

function luminance(hex: string): number {
  const [r, g, b] = channels(hex).map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)) as [
    number,
    number,
    number,
  ];
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/** WCAG contrast ratio between two opaque hex colors, from 1 to 21. */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x) as [number, number];
  return (hi + 0.05) / (lo + 0.05);
}
