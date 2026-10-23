// Conventions in docs/estilos.md. stylelint/stylelint.test.ts checks the key rules.

// Modules over cuadra/max-lines before the rule existed. Each task of the
// T26b–T26j block removes the ones of its feature; T26j deletes the list.
// A file never enters it: split the module instead.
const legacyLongModules = [
  'src/features/sales/CartLines.module.scss',
  'src/features/sales/CheckoutDialog.module.scss',
  'src/features/sales/PosPage.module.scss',
  'src/features/sales/Receipt.module.scss',
  'src/shared/ui/Toast/Toast.module.scss',
];

/** @type {import('stylelint').Config} */
export default {
  extends: ['stylelint-config-standard-scss'],
  plugins: ['stylelint-declaration-strict-value', './stylelint/maxLines.mjs'],
  rules: {
    // Class names in camelCase: the module exposes them as styles.priceCell.
    'selector-class-pattern': [
      '^[a-z][a-zA-Z0-9]*$',
      { message: (selector) => `La clase "${selector}" debe estar en camelCase (docs/estilos.md)` },
    ],
    'max-nesting-depth': 3,
    'declaration-no-important': true,
    'selector-max-id': 0,
    // Sass deprecated @import: use @use / @forward.
    'at-rule-disallowed-list': ['import'],
    // Shared styles go through mixins or a component, never `composes`.
    'property-disallowed-list': ['composes'],
    // :global is allowed only with a comment that justifies it (reviewed by hand).
    'selector-pseudo-class-no-unknown': [true, { ignorePseudoClasses: ['global'] }],
  },
  overrides: [
    {
      // In modules, every visual value comes from a semantic token.
      files: ['**/*.module.scss'],
      rules: {
        'color-no-hex': true,
        'color-named': 'never',
        'function-disallowed-list': ['rgb', 'rgba', 'hsl', 'hsla', 'hwb', 'lab', 'lch', 'oklab', 'oklch'],
        'scale-unlimited/declaration-strict-value': [
          ['/color$/', 'fill', 'stroke', 'z-index', 'box-shadow'],
          { ignoreValues: ['currentcolor', 'transparent', 'inherit', 'initial', 'unset', 'none', 'auto'] },
        ],
        // Loose px are not allowed: only 1px borders and outlines.
        'unit-disallowed-list': [['px'], { ignoreProperties: { px: ['/^border/', '/^outline/'] } }],
        // One piece per module, like a component (docs/frontend.md).
        'cuadra/max-lines': 100,
      },
    },
    { files: legacyLongModules, rules: { 'cuadra/max-lines': null } },
  ],
};
