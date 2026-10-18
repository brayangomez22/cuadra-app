import { useQuery } from '@tanstack/react-query';
import clsx from 'clsx';
import { useId, type ComponentProps } from 'react';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { Alert, Button, IconAlert, Skeleton } from '@/shared/ui';
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
  const templates = useQuery({
    queryKey: ['auth', 'business-templates'],
    queryFn: async () => unwrap(await api.GET('/api/v1/business-templates')).items,
    staleTime: Infinity,
  });

  return (
    <div className={clsx(styles.root, className)}>
      <p id={labelId} className={styles.label}>
        Tipo de negocio
      </p>
      <p className={styles.hint}>Prepara sus unidades, categorías y campos de producto. Después puede cambiarlo todo.</p>
      {templates.isPending && (
        <div className={styles.options} aria-busy="true">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} shape="block" className={styles.placeholder} />
          ))}
        </div>
      )}
      {templates.isError && (
        <Alert tone="danger" title="No pudimos cargar los tipos de negocio">
          <span className={styles.retry}>
            {describeError(templates.error).message}
            <Button variant="secondary" onClick={() => void templates.refetch()}>
              Reintentar
            </Button>
          </span>
        </Alert>
      )}
      {templates.data && (
        <div
          role="radiogroup"
          aria-labelledby={labelId}
          aria-invalid={error ? true : undefined}
          aria-errormessage={error ? errorId : undefined}
          className={styles.options}
        >
          {templates.data.map((template) => (
            <label key={template.id} className={styles.option}>
              <input
                type="radio"
                className={styles.radio}
                value={template.id}
                aria-labelledby={`${id}-${template.id}-name`}
                aria-describedby={`${id}-${template.id}`}
                {...radio}
              />
              <span className={styles.text}>
                <span id={`${id}-${template.id}-name`} className={styles.name}>
                  {template.name}
                </span>
                <span id={`${id}-${template.id}`} className={styles.description}>
                  {template.description}
                </span>
              </span>
            </label>
          ))}
        </div>
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
