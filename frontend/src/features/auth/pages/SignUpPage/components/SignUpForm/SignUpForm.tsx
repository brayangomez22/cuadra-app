import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { Alert, Button, Field, Input } from '@/shared/ui';
import { signUpSchema, type SignUpValues } from '../../../../model/schemas';
import { useSignUp } from '../../hooks/useSignUp';
import { BusinessTypeField } from '../BusinessTypeField/BusinessTypeField';
import styles from './SignUpForm.module.scss';

const emptyValues: SignUpValues = {
  tenant_name: '',
  nit: '',
  name: '',
  email: '',
  password: '',
  business_template: '',
};

export function SignUpForm() {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<SignUpValues>({ resolver: zodResolver(signUpSchema), defaultValues: emptyValues });
  const { formError, submit } = useSignUp(setError);

  return (
    <form className={styles.form} noValidate onSubmit={handleSubmit(submit)}>
      {formError && (
        <Alert tone="danger" className={styles.wide}>
          {formError}
        </Alert>
      )}
      <Field label="Nombre de la empresa" error={errors.tenant_name?.message}>
        <Input autoComplete="organization" autoFocus {...register('tenant_name')} />
      </Field>
      <Field label="NIT" hint="Con dígito de verificación: 900123456-8" error={errors.nit?.message}>
        <Input inputMode="numeric" autoComplete="off" {...register('nit')} />
      </Field>
      <Field label="Su nombre" error={errors.name?.message}>
        <Input autoComplete="name" {...register('name')} />
      </Field>
      <Field label="Correo" hint="Con este correo ingresa a Cuadra" error={errors.email?.message}>
        <Input type="email" autoComplete="username" inputMode="email" {...register('email')} />
      </Field>
      <Field label="Contraseña" hint="Mínimo 8 caracteres" error={errors.password?.message}>
        <Input type="password" autoComplete="new-password" {...register('password')} />
      </Field>
      <BusinessTypeField
        className={styles.wide}
        error={errors.business_template?.message}
        {...register('business_template')}
      />
      <Button type="submit" size="lg" className={styles.submit} loading={isSubmitting}>
        Crear cuenta
      </Button>
    </form>
  );
}
