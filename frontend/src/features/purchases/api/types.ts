import type { components, operations } from '@/shared/api/schema';

export type Supplier = components['schemas']['Supplier'];
export type SupplierInput = components['schemas']['SupplierInput'];
export type SupplierQuery = NonNullable<operations['listSuppliers']['parameters']['query']>;
export type SupplierStatus = NonNullable<SupplierQuery['status']>;
export type Purchase = components['schemas']['Purchase'];
export type PurchaseInput = components['schemas']['PurchaseInput'];
export type PurchaseLine = components['schemas']['PurchaseLine'];
export type PurchaseTax = components['schemas']['PurchaseTax'];
export type PurchaseSummary = components['schemas']['PurchaseSummary'];
export type PurchasePayment = components['schemas']['PurchasePayment'];
export type PurchaseQuery = NonNullable<operations['listPurchases']['parameters']['query']>;
export type PurchaseStatusFilter = NonNullable<PurchaseQuery['status']>;
