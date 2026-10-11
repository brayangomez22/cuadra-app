// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass-embedded';
import { contrast, customProperties, mediaBlock, withoutMedia, type Tokens } from './css';

// Tokens are checked on the compiled CSS, the same output the browser receives.
const stylesDir = fileURLToPath(new URL('../src/styles', import.meta.url));
const estilosMd = fileURLToPath(new URL('../../docs/estilos.md', import.meta.url));

const LIGHT = ':root';
const DARK = ":root[data-theme='dark']";
const DARK_SYSTEM = ":root:not([data-theme='light'])";

const TEXT = 4.5; // WCAG 1.4.3, normal text.
const NON_TEXT = 3; // WCAG 1.4.11, focus indicators and control borders.

// Every foreground/background pair the UI is allowed to use. A new semantic
// color token needs its pairs here before a component uses it.
const pairs: [fg: string, bg: string, min: number][] = [
  ['--color-text-default', '--color-bg-canvas', TEXT],
  ['--color-text-default', '--color-bg-surface', TEXT],
  ['--color-text-default', '--color-bg-subtle', TEXT],
  ['--color-text-muted', '--color-bg-canvas', TEXT],
  ['--color-text-muted', '--color-bg-surface', TEXT],
  ['--color-text-muted', '--color-bg-subtle', TEXT],
  ['--color-text-on-action', '--color-action-primary', TEXT],
  ['--color-text-on-action', '--color-action-primary-hover', TEXT],
  ['--color-text-on-action', '--color-action-primary-active', TEXT],
  ['--color-text-on-danger', '--color-action-danger', TEXT],
  ['--color-text-on-danger', '--color-action-danger-hover', TEXT],
  ['--color-feedback-danger', '--color-bg-surface', TEXT],
  ['--color-feedback-danger', '--color-feedback-danger-subtle', TEXT],
  ['--color-feedback-success', '--color-bg-surface', TEXT],
  ['--color-feedback-success', '--color-feedback-success-subtle', TEXT],
  ['--color-feedback-warning', '--color-bg-surface', TEXT],
  ['--color-feedback-warning', '--color-feedback-warning-subtle', TEXT],
  ['--color-feedback-info', '--color-bg-surface', TEXT],
  ['--color-feedback-info', '--color-feedback-info-subtle', TEXT],
  ['--color-focus-ring', '--color-bg-canvas', NON_TEXT],
  ['--color-focus-ring', '--color-bg-surface', NON_TEXT],
  ['--color-border-strong', '--color-bg-canvas', NON_TEXT],
  ['--color-border-strong', '--color-bg-surface', NON_TEXT],
  ['--color-border-accent', '--color-bg-surface', NON_TEXT],
];

async function compileMain() {
  const { css } = await sass.compileAsync(`${stylesDir}/main.scss`, { style: 'expanded' });
  return css;
}

function compileWithAbstracts(source: string) {
  return sass.compileStringAsync(`@use 'abstracts' as *;\n${source}`, { loadPaths: [stylesDir], style: 'expanded' });
}

function colorTokens(tokens: Tokens): string[] {
  return [...tokens.keys()].filter((name) => name.startsWith('--color-')).sort();
}

function failingPairs(tokens: Tokens, min: number) {
  return pairs
    .filter(([, , pairMin]) => pairMin === min)
    .map(([fg, bg]) => {
      const [fgValue, bgValue] = [tokens.get(fg), tokens.get(bg)];
      if (!fgValue || !bgValue) return `${fg} / ${bg}: token sin definir`;
      const ratio = contrast(fgValue, bgValue);
      return ratio < min ? `${fg} / ${bg}: ${ratio.toFixed(2)}:1` : null;
    })
    .filter(Boolean);
}

// Truncated, not rounded: the table never claims more contrast than there is.
const ratioText = (ratio: number) => `${(Math.floor(ratio * 100) / 100).toFixed(2)}:1`;

function contrastTable(themes: [name: string, tokens: Tokens][]): string {
  const rows = themes.flatMap(([name, tokens]) =>
    pairs.map(([fg, bg, min]) => {
      const ratio = contrast(tokens.get(fg) ?? '#000', tokens.get(bg) ?? '#000');
      return `| ${name} | \`${fg}\` | \`${bg}\` | ${ratioText(ratio)} | ${min}:1 |`;
    }),
  );
  return ['| Tema | Primer plano | Fondo | Contraste | Mínimo |', '|---|---|---|---|---|', ...rows].join('\n');
}

describe('tokens de diseño', () => {
  let css: string;
  let light: Tokens;
  let dark: Tokens;

  beforeAll(async () => {
    css = await compileMain();
    light = customProperties(withoutMedia(css), LIGHT);
    dark = customProperties(withoutMedia(css), DARK);
  });

  it('todos los pares texto/fondo del tema claro cumplen 4.5:1', () => {
    expect(failingPairs(light, TEXT)).toEqual([]);
  });

  it('todos los pares texto/fondo del tema oscuro cumplen 4.5:1', () => {
    expect(failingPairs(dark, TEXT)).toEqual([]);
  });

  it('el anillo de foco y los bordes de controles cumplen 3:1 contra su fondo', () => {
    expect(failingPairs(light, NON_TEXT)).toEqual([]);
    expect(failingPairs(dark, NON_TEXT)).toEqual([]);
  });

  it('el tema oscuro define exactamente los mismos tokens de color que el claro', () => {
    expect(colorTokens(light).length).toBeGreaterThan(0);
    expect(colorTokens(dark)).toEqual(colorTokens(light));
  });

  it('el tema del sistema operativo aplica los mismos valores que data-theme="dark"', () => {
    const system = customProperties(mediaBlock(css, '(prefers-color-scheme: dark)'), DARK_SYSTEM);
    expect(system.size).toBeGreaterThan(0);
    expect(Object.fromEntries(system)).toEqual(Object.fromEntries(dark));
  });

  it('main.scss compila y expone las escalas de espacio, tipografía, radios, sombras, movimiento y capas', () => {
    const expected = [
      ...['1', '2', '3', '4', '5', '6', '8', '10', '12', '16'].map((n) => `--space-${n}`),
      ...['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl', 'display'].map((s) => `--font-size-${s}`),
      '--line-height-tight',
      '--line-height-normal',
      ...['regular', 'medium', 'semibold', 'bold', 'heavy'].map((w) => `--font-weight-${w}`),
      '--font-family-sans',
      '--font-family-mono',
      ...['sm', 'md', 'lg', 'full'].map((r) => `--radius-${r}`),
      ...['sm', 'md', 'lg'].map((s) => `--shadow-${s}`),
      '--duration-fast',
      '--duration-normal',
      '--ease-standard',
      ...['dropdown', 'sticky', 'modal', 'toast'].map((z) => `--z-${z}`),
      '--size-control',
      '--size-row',
      '--size-touch-min',
    ];
    expect(expected.filter((name) => !light.has(name))).toEqual([]);
  });

  it('el texto base mide al menos 16px', () => {
    expect(light.get('--font-size-md')).toBe('1rem');
  });

  it('en pantallas táctiles las filas y los controles crecen', () => {
    const touch = customProperties(mediaBlock(css, '(pointer: coarse)'), LIGHT);
    const rem = (v: string | undefined) => parseFloat(v ?? '0');
    expect(rem(touch.get('--size-row'))).toBeGreaterThan(rem(light.get('--size-row')));
    expect(rem(touch.get('--size-control'))).toBeGreaterThan(rem(light.get('--size-control')));
  });

  it('respond-to falla al compilar con un breakpoint inexistente', async () => {
    await expect(compileWithAbstracts('.a { @include respond-to(xxl) { display: none; } }')).rejects.toThrow(
      /breakpoint/i,
    );
  });

  it('respond-to genera una media query mobile-first con min-width', async () => {
    const { css: out } = await compileWithAbstracts('.a { @include respond-to(md) { display: none; } }');
    expect(out).toMatch(/@media \(min-width: 48rem\)/);
  });

  it('touch-target asegura 44px de alto y ancho mínimos', async () => {
    const { css: out } = await compileWithAbstracts('.a { @include touch-target; }');
    expect(out).toContain('min-block-size: 2.75rem');
    expect(out).toContain('min-inline-size: 2.75rem');
  });

  it('hover solo aplica en dispositivos con puntero que permite hover', async () => {
    const { css: out } = await compileWithAbstracts('.a { @include hover { color: inherit; } }');
    expect(out).toMatch(/@media \(hover: hover\)\s*\{\s*\.a:hover/);
  });

  it('la tabla de contraste de docs/estilos.md coincide con los tokens', async () => {
    const doc = await readFile(estilosMd, 'utf8');
    expect(doc).toContain(contrastTable([['claro', light], ['oscuro', dark]]));
  });
});
