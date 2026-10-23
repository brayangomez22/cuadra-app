import type { AttributeDefinition } from '../../../../api/types';
import { AttributeField } from '../AttributeField/AttributeField';
import { FormGrid, FormSection } from '../FormSection/FormSection';

/** "Atributos": the fields of the attributes the tenant defines for its products. */
export function AttributesSection({ definitions }: { definitions: AttributeDefinition[] }) {
  return (
    <FormSection title="Atributos">
      <FormGrid>
        {definitions.map((definition) => (
          <AttributeField key={definition.key} definition={definition} />
        ))}
      </FormGrid>
    </FormSection>
  );
}
