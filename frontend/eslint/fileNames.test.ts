// @vitest-environment node
import { RuleTester } from 'eslint';
import { fileNames } from './fileNames.mjs';

const tester = new RuleTester({ languageOptions: { ecmaVersion: 2023, sourceType: 'module' } });
const code = 'export const x = 1;';
const error = [{ messageId: 'genericName' }];

// RuleTester runs each case as a test of its own.
tester.run('cuadra/file-names', fileNames, {
  valid: [
    {
      name: 'permite un archivo con nombre de lo que hace',
      filename: '/repo/frontend/src/features/sales/model/cart/lineTotals.ts',
      code,
    },
    { name: 'permite su test', filename: '/repo/frontend/src/features/sales/model/cart/lineTotals.test.ts', code },
    {
      name: 'no confunde un nombre que solo empieza igual',
      filename: '/repo/frontend/src/shared/format/utilization.ts',
      code,
    },
  ],
  invalid: [
    { name: 'rechaza utils.ts', filename: '/repo/frontend/src/features/catalog/utils.ts', code, errors: error },
    { name: 'rechaza helpers.tsx', filename: '/repo/frontend/src/shared/ui/helpers.tsx', code, errors: error },
    { name: 'rechaza common.ts', filename: '/repo/frontend/src/shared/common.ts', code, errors: error },
    {
      name: 'rechaza el test de un utils',
      filename: '/repo/frontend/src/features/sales/utils.test.ts',
      code,
      errors: error,
    },
  ],
});
