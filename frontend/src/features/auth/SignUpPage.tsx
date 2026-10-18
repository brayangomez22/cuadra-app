import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation } from 'react-router';
import { api } from '@/shared/api/client';
import { errorCode, errorMessage, NETWORK_ERROR_MESSAGE } from '@/shared/api/errors';
import { Alert, Button, Field, Input } from '@/shared/ui';
import { AuthLink, AuthPage } from './AuthPage';
import { BusinessTypeField } from './BusinessTypeField';
import { signUpSchema, type SignUpValues } from './schemas';
import { useAuth } from './useAuth';
import styles from './SignUpPage.module.scss';

// The field each 422 code of POST /auth/signup belongs to (api/openapi.yaml).
const fieldByErrorCode: Record<string, keyof SignUpValues> = {
  invalid_tenant_name: 'tenant_name',
  invalid_nit: 'nit',
  invalid_nit_check_digit: 'nit',
  invalid_user_name: 'name',
  invalid_email: 'email',
  password_too_short: 'password',
  password_too_long: 'password',
  unknown_business_template: 'business_template',
};

export function SignUpPage() {
  const { startSession } = useAuth();
  const location = useLocation();
  const [formError, setFormError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(signUpSchema),
    defaultValues: { tenant_name: '', nit: '', name: '', email: '', password: '', business_template: '' },
  });

  async function signUp(values: SignUpValues) {
    setFormError(null);
    try {
      const { data, error } = await api.POST('/api/v1/auth/signup', { body: values });
      if (data) {
        startSession(data);
        return;
      }
      const message = errorMessage(error, 'No pudimos crear la cuenta. Intente de nuevo.');
      const field = fieldByErrorCode[errorCode(error) ?? ''];
      if (field) setError(field, { message }, { shouldFocus: true });
      else setFormError(message);
    } catch {
      setFormError(NETWORK_ERROR_MESSAGE);
    }
  }

  return (
    <AuthPage
      size="wide"
      title="Cree la cuenta de su negocio"
      description="Usted queda como dueño y después puede agregar a sus empleados."
      footer={
        <>
          ¿Ya tiene cuenta?{' '}
          <AuthLink to="/ingresar" state={location.state}>
            Ingrese
          </AuthLink>
        </>
      }
    >
      <form className={styles.form} noValidate onSubmit={handleSubmit(signUp)}>
        {formError && (
          <Alert tone="danger" className={styles.wide}>
            {formError}
          </Alert>
        )}
        <Field label="Nombre de la empresa" error={errors.tenant_name?.message}>
          <Input autoComplete="organization" autoFocus {...register('tenant_name')} />
        </Field>
        <Field label="NIT" hint="Con dígito de verificación: 890903938-8" error={errors.nit?.message}>
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
    </AuthPage>
  );
}
