// @vitest-environment node
import { RuleTester } from 'eslint';
import { featureBoundaries } from './featureBoundaries.mjs';

const tester = new RuleTester({ languageOptions: { ecmaVersion: 2023, sourceType: 'module' } });
const inCatalog = '/repo/frontend/src/features/catalog/pages/ProductFormPage/ProductFormPage.tsx';
const error = [{ messageId: 'crossFeature' }];

// RuleTester runs each case as a test of its own.
tester.run('cuadra/feature-boundaries', featureBoundaries, {
  valid: [
    {
      name: 'permite un import relativo dentro de la misma feature',
      filename: inCatalog,
      code: "import { x } from '../../api/products';",
    },
    {
      name: 'permite import() relativo dentro de la feature',
      filename: inCatalog,
      code: "const pages = () => import('../../pages');",
    },
    {
      name: 'ignora los imports por alias y de paquetes',
      filename: inCatalog,
      code: "import { useAuth } from '@/features/auth'; import { useQuery } from '@tanstack/react-query';",
    },
    {
      name: 'permite que un archivo fuera de features importe relativo fuera de features',
      filename: '/repo/frontend/src/app/router.tsx',
      code: "import { paths } from './routes/paths';",
    },
  ],
  invalid: [
    {
      name: 'rechaza un import relativo que entra a otra feature',
      filename: inCatalog,
      code: "import { useLocations } from '../../../inventory/api/locations';",
      errors: error,
    },
    {
      name: 'rechaza un import() relativo a otra feature',
      filename: inCatalog,
      code: "const pages = () => import('../../../sales/pages');",
      errors: error,
    },
    {
      name: 'rechaza reexportar desde otra feature',
      filename: inCatalog,
      code: "export { useCan } from '../../../auth/session/useCan';",
      errors: error,
    },
    {
      name: 'rechaza que app/ entre relativo a una feature',
      filename: '/repo/frontend/src/app/router.tsx',
      code: "import { x } from '../features/catalog/pages';",
      errors: error,
    },
  ],
});
