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
  'src/features/inventory/CreateLocationDialog.tsx': ['max-lines-per-function'],
  'src/features/inventory/KardexPage.tsx': ['complexity', 'max-lines', 'max-lines-per-function'],
  'src/features/inventory/ReceiptPage.test.tsx': ['max-lines'],
  'src/features/inventory/ReceiptPage.tsx': [
    '@typescript-eslint/no-unnecessary-condition',
    'complexity',
    'max-lines',
    'max-lines-per-function',
  ],
  'src/features/inventory/StockPage.tsx': ['complexity', 'max-lines', 'max-lines-per-function'],
  'src/features/sales/CartLines.tsx': [
    '@typescript-eslint/no-unnecessary-condition',
    'complexity',
    'max-lines',
    'max-lines-per-function',
  ],
  'src/features/sales/CashPage.test.tsx': ['max-lines'],
  'src/features/sales/CashPage.tsx': [
    '@typescript-eslint/no-misused-promises',
    'complexity',
    'max-lines',
    'max-lines-per-function',
  ],
  'src/features/sales/CashReport.tsx': ['max-lines-per-function'],
  'src/features/sales/CashReportPage.tsx': [
    '@typescript-eslint/prefer-optional-chain',
    'complexity',
    'max-lines-per-function',
  ],
  'src/features/sales/CheckoutDialog.tsx': ['@typescript-eslint/no-deprecated', 'max-lines', 'max-lines-per-function'],
  'src/features/sales/CloseCashPage.tsx': ['@typescript-eslint/no-deprecated', 'max-lines', 'max-lines-per-function'],
  'src/features/sales/PosPage.test.tsx': [
    '@typescript-eslint/no-unsafe-member-access',
    '@typescript-eslint/no-unsafe-return',
    '@typescript-eslint/unbound-method',
    'max-lines',
  ],
  'src/features/sales/PosPage.tsx': [
    '@typescript-eslint/no-unnecessary-condition',
    'complexity',
    'max-lines',
    'max-lines-per-function',
  ],
  'src/features/sales/ProductSearch.tsx': [
    '@typescript-eslint/no-deprecated',
    'complexity',
    'max-lines',
    'max-lines-per-function',
  ],
  'src/features/sales/Receipt.test.tsx': ['max-lines', 'max-params'],
  'src/features/sales/Receipt.tsx': ['max-lines-per-function'],
  'src/features/sales/cart.test.ts': ['max-lines'],
  'src/features/sales/cart.ts': ['complexity', 'max-lines'],
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
