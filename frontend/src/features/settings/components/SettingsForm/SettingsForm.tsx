import type { ReactNode, SubmitEventHandler } from 'react';
import { Alert, Button, FormSection } from '@/shared/ui';
import styles from './SettingsForm.module.scss';

type Props = {
  title: string;
  hint?: ReactNode;
  /** The button's name: each block of a page saves on its own. */
  submitLabel: string;
  onSubmit: SubmitEventHandler<HTMLFormElement>;
  saving: boolean;
  /** The message of an error that belongs to no field. */
  error?: string | null;
  children: ReactNode;
};

/** A block of settings that saves on its own: its title, fields, error and button. */
export function SettingsForm({ title, hint, submitLabel, onSubmit, saving, error, children }: Props) {
  return (
    <FormSection title={title} hint={hint}>
      <form className={styles.root} noValidate onSubmit={onSubmit}>
        {error && <Alert tone="danger">{error}</Alert>}
        {children}
        <Button type="submit" className={styles.submit} loading={saving}>
          {submitLabel}
        </Button>
      </form>
    </FormSection>
  );
}
