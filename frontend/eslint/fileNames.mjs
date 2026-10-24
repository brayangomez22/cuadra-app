import path from 'node:path';

// cuadra/file-names rejects the generic file names docs/frontend.md forbids
// (utils, helpers, common, misc): a file is named after what it does
// (lineTotals.ts, toProductInput.ts), so it can be found without opening it.

const generic = /^(utils?|helpers?|common|misc)$/i;

/** @type {import('eslint').Rule.RuleModule} */
export const fileNames = {
  meta: {
    type: 'suggestion',
    messages: {
      genericName: '"{{name}}" no dice qué hace el archivo: nómbralo por su propósito (docs/frontend.md, sección 2).',
    },
    schema: [],
  },
  create(context) {
    const name = path.basename(context.filename).replace(/(\.test)?\.[^.]+$/, '');
    return {
      Program(node) {
        if (generic.test(name)) context.report({ node, messageId: 'genericName', data: { name } });
      },
    };
  },
};
