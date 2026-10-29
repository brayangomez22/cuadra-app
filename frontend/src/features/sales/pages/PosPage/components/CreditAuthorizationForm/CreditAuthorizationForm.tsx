import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { formatMoney } from '@/shared/format';
import { Alert, Button, Field, Input } from '@/shared/ui';
import {
  creditAuthorizationSchema,
  type CreditAuthorizationValues,
} from '../../../../model/payments/creditAuthorizationSchema';
import styles from './CreditAuthorizationForm.module.scss';

type Props = {
  /** What the customer had left of their limit when the sale went over it. */
  available: string;
  onAuthorize: (credentials: CreditAuthorizationValues) => void;
  onBack: () => void;
  pending: boolean;
  /** Why the API rejected the authorized sale, in Spanish. */
  error?: string;
};

/**
 * An administrator authorizes, with their email and password, a credit sale
 * over the customer's limit. The password is cleared once sent: it is never
 * kept on screen.
 */
export function CreditAuthorizationForm({ available, onAuthorize, onBack, pending, error }: Props) {
  const [sent, setSent] = useState(false);
  const form = useForm({
    resolver: zodResolver(creditAuthorizationSchema),
    defaultValues: { email: '', password: '' },
  });
  const { errors } = form.formState;
  const submit = form.handleSubmit((credentials) => {
    setSent(true);
    form.resetField('password');
    onAuthorize(credentials);
  });
  return (
    <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
      <p>Le quedan {formatMoney(available)} de cupo.</p>
      <Field label="Correo del administrador" error={errors.email?.message}>
        <Input type="email" autoComplete="off" autoFocus {...form.register('email')} />
      </Field>
      <Field label="Contraseña" error={errors.password?.message}>
        <Input type="password" autoComplete="off" {...form.register('password')} />
      </Field>
      {sent && error && !pending && <Alert tone="danger">{error}</Alert>}
      <div className={styles.footer}>
        <Button variant="secondary" onClick={onBack}>
          Volver a los pagos
        </Button>
        <Button type="submit" size="lg" loading={pending}>
          Autorizar y vender
        </Button>
      </div>
    </form>
  );
}
