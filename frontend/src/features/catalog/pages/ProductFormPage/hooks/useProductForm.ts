import { zodResolver } from '@hookform/resolvers/zod';
import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { productDefaults } from '../../../model/productDefaults';
import { productSchema, type ProductFormInput, type ProductFormValues } from '../../../model/productSchema';
import type { ProductFormData } from './useProductFormData';

/** The product form with the tenant's attributes, starting from the product's values (or a new one's). */
export function useProductForm({ product, units, definitions, axes, defaultTaxRate }: ProductFormData) {
  const schema = useMemo(() => productSchema(definitions), [definitions]);
  return useForm<ProductFormInput, unknown, ProductFormValues>({
    resolver: zodResolver(schema),
    defaultValues: productDefaults({ product, units, definitions, axes, defaultTaxRate }),
  });
}

export type ProductFormApi = ReturnType<typeof useProductForm>;
