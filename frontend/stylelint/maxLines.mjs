import stylelint from 'stylelint';

// cuadra/max-lines caps a style module like ESLint's max-lines caps a
// component (docs/frontend.md): a module past the limit styles more than one
// piece. Blank lines and comments do not count.

const ruleName = 'cuadra/max-lines';
const messages = stylelint.utils.ruleMessages(ruleName, {
  rejected: (lines, max) => `El módulo tiene ${lines} líneas y el máximo es ${max}: divídelo (docs/frontend.md)`,
});

function codeLines(css) {
  return css
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .split('\n')
    .filter((line) => {
      const text = line.trim();
      return text !== '' && !text.startsWith('//');
    }).length;
}

/** @type {import('stylelint').Rule<number>} */
const rule = (max) => (root, result) => {
  if (!stylelint.utils.validateOptions(result, ruleName, { actual: max, possible: [(n) => Number.isInteger(n)] })) {
    return;
  }
  const lines = codeLines(root.source?.input.css ?? '');
  if (lines <= max) return;
  stylelint.utils.report({ result, ruleName, node: root, message: messages.rejected(lines, max) });
};

rule.ruleName = ruleName;
rule.messages = messages;

export default stylelint.createPlugin(ruleName, rule);
