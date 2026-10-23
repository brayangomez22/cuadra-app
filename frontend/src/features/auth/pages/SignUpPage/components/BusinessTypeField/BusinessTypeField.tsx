import clsx from 'clsx';
import { useId, type ComponentProps } from 'react';
import { describeError } from '@/shared/api/errors';
import { Alert, Button, IconAlert, Inline } from '@/shared/ui';
import { useBusinessTemplates } from '../../../../api/businessTemplates';
import { BusinessTypeOptions, BusinessTypeOptionsSkeleton } from '../BusinessTypeOptions/BusinessTypeOptions';
import styles from './BusinessTypeField.module.scss';

type Props = Omit<ComponentProps<'input'>, 'type' | 'id'> & {
  error?: string;
  className?: string;
};

/**
 * The business type a tenant signs up with: one card per template of GET
 * /business-templates, as native radios. It takes the props of
 * react-hook-form's register(), shared by every radio.
 */
export function BusinessTypeField({ error, className, ...radio }: Props) {
  const id = useId();
  const labelId = `${id}-label`;
  const errorId = `${id}-error`;
  const templates = useBusinessTemplates();

  return (
    <div className={clsx(styles.root, className)}>
      <p id={labelId} className={styles.label}>
        Tipo de negocio
      </p>
      <p className={styles.hint}>
        Prepara sus unidades, categorías y campos de producto. Después puede cambiarlo todo.
      </p>
      {templates.isPending && <BusinessTypeOptionsSkeleton />}
      {templates.isError && (
        <Alert tone="danger" title="No pudimos cargar los tipos de negocio">
          <Inline as="span" gap="3" wrap>
            {describeError(templates.error).message}
            <Button variant="secondary" onClick={() => void templates.refetch()}>
              Reintentar
            </Button>
          </Inline>
        </Alert>
      )}
      {templates.data && (
        <BusinessTypeOptions
          templates={templates.data}
          id={id}
          labelId={labelId}
          errorId={error ? errorId : undefined}
          radio={radio}
        />
      )}
      {error && (
        <p id={errorId} className={styles.error}>
          <IconAlert className={styles.errorIcon} />
          <span>{error}</span>
        </p>
      )}
    </div>
  );
}
