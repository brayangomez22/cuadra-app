import path from 'node:path';

// cuadra/feature-boundaries rejects a relative import that lands in another
// feature (../../inventory/api): no-restricted-imports only sees the text of
// the path, so it catches the aliases (@/features/x/...) but not where a
// relative path goes. A feature is reached only through @/features/x.

// The feature a file lives in, or undefined outside src/features/.
function featureOf(file) {
  return /\/src\/features\/([^/]+)\//.exec(file.split(path.sep).join('/'))?.[1];
}

/** @type {import('eslint').Rule.RuleModule} */
export const featureBoundaries = {
  meta: {
    type: 'problem',
    messages: {
      crossFeature: 'Este import entra a la feature {{target}}: impórtala por su index.ts (@/features/{{target}}).',
    },
    schema: [],
  },
  create(context) {
    const own = featureOf(context.filename);
    function check(source) {
      if (typeof source?.value !== 'string' || !source.value.startsWith('.')) return;
      const target = featureOf(`${path.resolve(path.dirname(context.filename), source.value)}/`);
      if (target && target !== own) context.report({ node: source, messageId: 'crossFeature', data: { target } });
    }
    return {
      ImportDeclaration: (node) => check(node.source),
      ImportExpression: (node) => check(node.source),
      ExportNamedDeclaration: (node) => check(node.source),
      ExportAllDeclaration: (node) => check(node.source),
    };
  },
};
