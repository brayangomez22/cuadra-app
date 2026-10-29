// The fixtures several features share: the products (catalog, inventory and
// sales test with them), their units, the tenant's attributes and axes
// (catalog and settings), the locations and the customers and their segments
// (customers, settings and sales) and their credit (receivables and sales). A
// fixture only one feature uses lives in that feature's testing/.
export { attributes } from './attributes';
export { axes } from './axes';
export { aCredit, buckets, noCredit, statement } from './credit';
export { aBoutique, aBuilder, customerIds, customerInput, searchCustomers, segmentIds, segments } from './customers';
export { aGarment, variantBarcode, variantId } from './garment';
export { ids } from './ids';
export { aLocation } from './locations';
export { aCement, aProduct, withoutCost } from './products';
export { page, plain, query } from './requests';
export type { Location, Product, ProductVariant, VariantUnit } from './types';
export { unitOf, units } from './units';
export { aVariantUnit, boxOf12, packOf3, rollOf100, withUnits } from './variantUnits';
