// @vitest-environment node
import { fileURLToPath } from 'node:url';
import stylelint from 'stylelint';

// The fixtures live outside src/, so `npm run lint:styles` does not lint them.
const fixture = (name: string) => fileURLToPath(new URL(`./fixtures/${name}`, import.meta.url));
const configFile = fileURLToPath(new URL('../stylelint.config.mjs', import.meta.url));

async function ruleViolations(name: string) {
  const { results } = await stylelint.lint({ files: fixture(name), configFile });
  const [result] = results;
  if (!result) throw new Error(`stylelint did not lint ${name}`);
  return result.warnings.map((w) => w.rule);
}

describe('stylelint', () => {
  it('rechaza un color hex dentro de un .module.scss', async () => {
    expect(await ruleViolations('HexColor.module.scss')).toContain('color-no-hex');
  });

  it('rechaza una clase que no está en camelCase', async () => {
    expect(await ruleViolations('KebabClass.module.scss')).toContain('selector-class-pattern');
  });

  it('rechaza px sueltos, pero acepta bordes de 1px', async () => {
    expect(await ruleViolations('LoosePx.module.scss')).toEqual(['unit-disallowed-list']);
  });

  it('acepta un módulo que solo usa tokens', async () => {
    expect(await ruleViolations('Valid.module.scss')).toEqual([]);
  });
});
