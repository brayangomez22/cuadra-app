import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];
export type Product = Schemas['Product'];
export type ProductVariant = Schemas['ProductVariant'];
export type UnitOfMeasure = Schemas['UnitOfMeasure'];
export type VariantUnit = Schemas['VariantUnit'];
export type Location = Schemas['Location'];
