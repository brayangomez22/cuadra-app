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
  /** The block has changes not saved yet: only then its button is the primary one. */
  changed: boolean;
  /** The message of an error that belongs to no field. */
  error?: string | null;
  children: ReactNode;
};

/**
 * A block of settings that saves on its own: its title, fields, error and
 * button. With several blocks on a page, the orange button marks the one with
 * changes to save, instead of four equal buttons.
 */
export function SettingsForm({ title, hint, submitLabel, onSubmit, saving, changed, error, children }: Props) {
  return (
    <FormSection title={title} hint={hint}>
      <form className={styles.root} noValidate onSubmit={onSubmit}>
        {error && <Alert tone="danger">{error}</Alert>}
        {children}
        <Button type="submit" variant={changed ? 'primary' : 'secondary'} className={styles.submit} loading={saving}>
          {submitLabel}
        </Button>
      </form>
    </FormSection>
  );
}
