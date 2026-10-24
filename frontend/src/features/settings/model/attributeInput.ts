import type { AttributeDefinition } from '@/features/catalog';
import type { components } from '@/shared/api/schema';
import type { AttributeValues } from './attributeSchema';
import { joinLines, parseLines } from './lines';

type Schemas = components['schemas'];

const newAttribute: AttributeValues = { label: '', key: '', type: 'text', options: '', required: false };

/** The form of an attribute: empty to add one, or its values to edit it. */
export function toAttributeValues(attribute: AttributeDefinition | null): AttributeValues {
  if (!attribute) return newAttribute;
  const { label, key, type, options, required } = attribute;
  return { label, key, type, options: joinLines(options), required };
}

// Only a list has options: one per line of the text box.
function optionsOf(values: AttributeValues) {
  return values.type === 'list' ? { options: parseLines(values.options) } : {};
}

/** The body that creates an attribute from its form. */
export function toAttributeCreate(values: AttributeValues): Schemas['AttributeDefinitionInput'] {
  const { key, label, type, required } = values;
  return { key, label, type, ...optionsOf(values), required };
}

/** The body that updates an attribute: its key and type never change. */
export function toAttributeUpdate(values: AttributeValues): Schemas['AttributeDefinitionUpdate'] {
  return { label: values.label, ...optionsOf(values), required: values.required };
}
