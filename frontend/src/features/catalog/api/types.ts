import type { components, operations } from '@/shared/api/schema';

export type Product = components['schemas']['Product'];
export type ProductVariant = components['schemas']['ProductVariant'];
export type VariantUnit = components['schemas']['VariantUnit'];
export type VariantUnitInput = components['schemas']['VariantUnitInput'];
export type ProductInput = components['schemas']['ProductInput'];
export type Category = components['schemas']['Category'];
export type UnitOfMeasure = components['schemas']['UnitOfMeasure'];
export type AttributeDefinition = components['schemas']['AttributeDefinition'];
export type VariantAxis = components['schemas']['VariantAxis'];
export type ProductQuery = NonNullable<operations['listProducts']['parameters']['query']>;
export type ProductStatus = NonNullable<ProductQuery['status']>;
export type ProductWithVariantsInput = components['schemas']['ProductWithVariantsInput'];
