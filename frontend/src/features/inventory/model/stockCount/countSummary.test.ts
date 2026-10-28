import { accuracyText, countedText } from './countSummary';

describe('accuracyText', () => {
  it.each([
    ['todas exactas', '1.0000', '100\u00a0%'],
    ['casi todas', '0.9500', '95\u00a0%'],
    ['con decimales', '0.8333', '83,33\u00a0%'],
    ['ninguna exacta', '0.0000', '0\u00a0%'],
  ])('muestra la exactitud: %s', (_, accuracy, text) => {
    expect(accuracyText(accuracy)).toBe(text);
  });

  it('sin líneas contadas no hay exactitud', () => {
    expect(accuracyText(undefined)).toBe('Sin contar');
  });
});

describe('countedText', () => {
  it.each([
    [{ lines: 120, counted_lines: 12 }, '12 de 120 productos contados'],
    [{ lines: 1, counted_lines: 1 }, '1 de 1 producto contado'],
    [{ lines: 0, counted_lines: 0 }, 'Aún no hay productos en el conteo'],
  ])('muestra el avance %#', (progress, text) => {
    expect(countedText(progress)).toBe(text);
  });
});
