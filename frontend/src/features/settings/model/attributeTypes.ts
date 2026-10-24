import type { AttributeDefinition } from '@/features/catalog';
import type { SelectOption } from '@/shared/ui';

type AttributeType = AttributeDefinition['type'];

const typeLabels: Record<AttributeType, string> = {
  text: 'Texto',
  number: 'Número',
  list: 'Lista',
  boolean: 'Sí/no',
};

export const typeOptions: SelectOption[] = (Object.keys(typeLabels) as AttributeType[]).map((value) => ({
  value,
  label: typeLabels[value],
}));

/** What a row says of an attribute's type: "Lista: Algodón, PVC". */
export function describeType(attribute: AttributeDefinition) {
  const label = typeLabels[attribute.type];
  return attribute.type === 'list' ? `${label}: ${attribute.options.join(', ')}` : label;
}
