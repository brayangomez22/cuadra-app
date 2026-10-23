// The catalog's screens, apart from index.ts so the router loads them (and
// the product form: zod, react-hook-form) only when someone opens them, not
// when another feature imports catalog to sell or count a variant.
export { ProductFormPage } from './pages/ProductFormPage/ProductFormPage';
export { ProductListPage } from './pages/ProductListPage/ProductListPage';
