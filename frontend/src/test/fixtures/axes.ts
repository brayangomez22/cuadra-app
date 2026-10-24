import type { components } from '@/shared/api/schema';

type VariantAxis = components['schemas']['VariantAxis'];

// aGarment's axes, as GET /variant-axes answers them.
export const axes: VariantAxis[] = [
  { id: '01920000-0000-7000-8000-000000000701', name: 'talla', values: ['S', 'M', 'L'] },
  { id: '01920000-0000-7000-8000-000000000702', name: 'color', values: ['Rojo', 'Azul'] },
];
