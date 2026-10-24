import type { components } from '@/shared/api/schema';

type AttributeDefinition = components['schemas']['AttributeDefinition'];

// The tenant's product attributes, as GET /product-attributes answers them.
export const attributes: AttributeDefinition[] = [
  {
    id: '01920000-0000-7000-8000-000000000801',
    key: 'marca',
    label: 'Marca',
    type: 'text',
    options: [],
    required: true,
  },
  {
    id: '01920000-0000-7000-8000-000000000802',
    key: 'material',
    label: 'Material',
    type: 'list',
    options: ['Algodón', 'PVC'],
    required: false,
  },
  {
    id: '01920000-0000-7000-8000-000000000803',
    key: 'peso',
    label: 'Peso',
    type: 'number',
    options: [],
    required: false,
  },
  {
    id: '01920000-0000-7000-8000-000000000804',
    key: 'importado',
    label: 'Importado',
    type: 'boolean',
    options: [],
    required: false,
  },
];
