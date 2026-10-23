import { planFiles } from './new';

describe('planFiles', () => {
  it('planea tsx, module.scss con .root y test para un componente de una feature', () => {
    const files = planFiles('component', 'catalog/pages/ProductFormPage/components/PriceSection');

    const dir = 'src/features/catalog/pages/ProductFormPage/components/PriceSection';
    expect(files.map((f) => f.path)).toEqual([
      `${dir}/PriceSection.tsx`,
      `${dir}/PriceSection.module.scss`,
      `${dir}/PriceSection.test.tsx`,
    ]);
    expect(files[0]?.content).toContain("import styles from './PriceSection.module.scss';");
    expect(files[0]?.content).toContain('export function PriceSection(');
    expect(files[0]?.content).toContain('className={styles.root}');
    expect(files[1]?.content).toContain('.root {');
    expect(files[2]?.content).toContain("describe('PriceSection'");
  });

  it('respeta las rutas de shared y app', () => {
    expect(planFiles('component', 'shared/ui/Stack')[0]?.path).toBe('src/shared/ui/Stack/Stack.tsx');
    expect(planFiles('hook', 'app/hooks/useLayout')[0]?.path).toBe('src/app/hooks/useLayout.ts');
  });

  it('planea un hook con su test', () => {
    const files = planFiles('hook', 'catalog/pages/ProductFormPage/hooks/useProductForm');

    expect(files.map((f) => f.path)).toEqual([
      'src/features/catalog/pages/ProductFormPage/hooks/useProductForm.ts',
      'src/features/catalog/pages/ProductFormPage/hooks/useProductForm.test.ts',
    ]);
    expect(files[0]?.content).toContain('export function useProductForm(');
  });

  it.each([
    ['component', 'catalog/components/priceSection', /PascalCase/],
    ['hook', 'catalog/hooks/productForm', /useCamelCase/],
    ['page', 'catalog/pages/X', /component o hook/],
    ['component', '', /ruta/],
  ])('rechaza %s en %s', (kind, path, message) => {
    expect(() => planFiles(kind, path)).toThrow(message);
  });
});
