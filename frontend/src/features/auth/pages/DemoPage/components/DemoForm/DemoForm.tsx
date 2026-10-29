import { useId } from 'react';
import { Alert, Button, Stack } from '@/shared/ui';
import type { DemoOption } from '../../../../api/types';
import { ChoiceCards } from '../../../../components/ChoiceCards/ChoiceCards';
import { roleOptions } from '../../../../model/roles';
import { useStartDemo } from '../../hooks/useStartDemo';
import styles from './DemoForm.module.scss';

type Props = { demos: DemoOption[] };

/** The business type and the role to enter the demo with. */
export function DemoForm({ demos }: Props) {
  const { form, formError, submit } = useStartDemo(demos);
  const typeId = useId();
  const roleId = useId();
  const {
    register,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  return (
    <form noValidate onSubmit={handleSubmit(submit)}>
      <Stack gap="5">
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Stack gap="2">
          <p id={`${typeId}-label`} className={styles.label}>
            Tipo de negocio
          </p>
          <ChoiceCards
            options={demos.map((d) => ({ value: d.business_template, label: d.name }))}
            id={typeId}
            labelId={`${typeId}-label`}
            radio={register('business_template')}
          />
        </Stack>
        <Stack gap="2">
          <p id={`${roleId}-label`} className={styles.label}>
            Entrar como
          </p>
          <ChoiceCards options={roleOptions} id={roleId} labelId={`${roleId}-label`} radio={register('role')} />
        </Stack>
        <Button type="submit" size="lg" className={styles.submit} loading={isSubmitting}>
          Entrar a la demo
        </Button>
      </Stack>
    </form>
  );
}
