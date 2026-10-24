// Files that broke the rules of eslint.config.js the day they arrived (T26b),
// with the rules each one still breaks: eslint.config.js turns off exactly
// those, in exactly that file.
//
// The list only shrinks. Each task of the T26b–T26j block deletes the entries
// of its feature (src/shared/ and src/test/ in T26c; src/app/ in T26j), and
// T26j deletes this file. A file never enters it, and an entry never gains a
// rule: split the file or fix the code instead (docs/frontend.md).
export const legacy = {
  'src/app/router.tsx': ['max-lines', 'no-restricted-syntax'],
  'src/features/dev/UiCatalogPage.tsx': ['max-lines', 'max-lines-per-function'],
  'src/features/settings/AttributesSettings.test.tsx': [
    '@typescript-eslint/no-unsafe-argument',
    '@typescript-eslint/no-unsafe-call',
    '@typescript-eslint/no-unsafe-member-access',
  ],
  'src/features/settings/AttributesSettings.tsx': ['complexity', 'max-lines', 'max-lines-per-function'],
  'src/features/settings/SettingsLayout.tsx': ['@typescript-eslint/no-unnecessary-condition'],
  'src/features/settings/UnitsSettings.tsx': ['complexity', 'max-lines', 'max-lines-per-function'],
  'src/features/settings/VariantAxesSettings.tsx': ['complexity', 'max-lines', 'max-lines-per-function'],
  'src/test/fixtures.ts': ['complexity', 'max-lines', 'max-params'],
};
