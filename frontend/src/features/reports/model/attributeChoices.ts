import type { components } from '@/shared/api/schema';

type AttributeDefinition = components['schemas']['AttributeDefinition'];

/** The values to choose from for filtering by an attribute, or null when the value is typed (text, number). */
export function attributeChoices(attribute: AttributeDefinition): { value: string; label: string }[] | null {
  switch (attribute.type) {
    case 'list':
      return attribute.options.map((option) => ({ value: option, label: option }));
    case 'boolean':
      // The API compares the stored JSON as text: true and false.
      return [
        { value: 'true', label: 'Sí' },
        { value: 'false', label: 'No' },
      ];
    default:
      return null;
  }
}
