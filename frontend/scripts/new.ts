// Creates a component or a hook following docs/frontend.md:
//
//   npm run new -- component catalog/pages/ProductFormPage/components/PriceSection
//   npm run new -- hook catalog/pages/ProductFormPage/hooks/useProductForm
//
// The path is relative to src/features/, unless it starts with shared/ or app/.
// Node runs this file as is (type stripping), so it uses only erasable syntax.
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

export type PlannedFile = { path: string; content: string };

const roots = ['shared/', 'app/'];

function srcPath(path: string): string {
  return roots.some((root) => path.startsWith(root)) ? `src/${path}` : `src/features/${path}`;
}

function component(name: string): string {
  return `import styles from './${name}.module.scss';

/** ${name}: say what it shows and why it is its own component. */
export function ${name}() {
  return <div className={styles.root} />;
}
`;
}

const moduleScss = `@use '@/styles/abstracts' as *;

.root {
  display: block;
}
`;

function componentTest(name: string): string {
  return `import { render } from '@testing-library/react';
import { ${name} } from './${name}';

describe('${name}', () => {
  it('se muestra', () => {
    const { container } = render(<${name} />);

    expect(container.firstChild).toBeInTheDocument();
  });
});
`;
}

function hook(name: string): string {
  return `/** ${name}: say what state it keeps and why. */
export function ${name}() {
  return {};
}
`;
}

function hookTest(name: string): string {
  return `import { renderHook } from '@testing-library/react';
import { ${name} } from './${name}';

describe('${name}', () => {
  it('devuelve su estado inicial', () => {
    const { result } = renderHook(() => ${name}());

    expect(result.current).toEqual({});
  });
});
`;
}

/** The files to create for a component or a hook, without touching the disk. */
export function planFiles(kind: string, path: string): PlannedFile[] {
  const clean = path.replace(/^\/+|\/+$/g, '');
  if (clean === '') throw new Error('Falta la ruta: npm run new -- component catalog/components/PriceTag');
  const name = clean.split('/').at(-1) ?? '';

  if (kind === 'component') {
    if (!/^[A-Z][A-Za-z0-9]*$/.test(name)) throw new Error(`"${name}" no es PascalCase (PriceSection).`);
    const dir = srcPath(clean);
    return [
      { path: `${dir}/${name}.tsx`, content: component(name) },
      { path: `${dir}/${name}.module.scss`, content: moduleScss },
      { path: `${dir}/${name}.test.tsx`, content: componentTest(name) },
    ];
  }
  if (kind === 'hook') {
    if (!/^use[A-Z][A-Za-z0-9]*$/.test(name)) throw new Error(`"${name}" no es useCamelCase (useProductForm).`);
    const file = srcPath(clean);
    return [
      { path: `${file}.ts`, content: hook(name) },
      { path: `${file}.test.ts`, content: hookTest(name) },
    ];
  }
  throw new Error(`"${kind}" no se genera: use component o hook.`);
}

function main(args: string[]): void {
  const [kind = '', path = ''] = args;
  const root = join(dirname(new URL(import.meta.url).pathname), '..');
  const files = planFiles(kind, path);
  const existing = files.filter((file) => existsSync(join(root, file.path)));
  if (existing.length > 0) throw new Error(`Ya existe: ${existing.map((file) => file.path).join(', ')}`);
  for (const file of files) {
    mkdirSync(dirname(join(root, file.path)), { recursive: true });
    writeFileSync(join(root, file.path), file.content);
    console.log(`creado ${file.path}`);
  }
}

if (import.meta.main) {
  try {
    main(process.argv.slice(2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  }
}
