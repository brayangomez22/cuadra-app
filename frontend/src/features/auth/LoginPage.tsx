import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation } from 'react-router';
import { api } from '@/shared/api/client';
import { errorMessage, NETWORK_ERROR_MESSAGE } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import { Alert, Button, Field, Input } from '@/shared/ui';
import { AuthLink, AuthPage } from './AuthPage';
import { loginSchema, type LoginValues } from './schemas';
import { useAuth } from './useAuth';
import styles from './LoginPage.module.scss';
import { paths } from '@/app/routes/paths';

type TenantOption = components['schemas']['TenantOption'];

// The same email and password are valid in several companies: the user picks one.
type TenantChoice = { credentials: LoginValues; tenants: TenantOption[] };

export function LoginPage() {
  const { startSession } = useAuth();
  const location = useLocation();
  const [formError, setFormError] = useState<string | null>(null);
  const [choice, setChoice] = useState<TenantChoice | null>(null);
  const [pendingTenant, setPendingTenant] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(loginSchema), defaultValues: { email: '', password: '' } });

  // A successful login stores the session; GuestOnly then leads to the app.
  async function login(credentials: LoginValues, tenantId?: string) {
    setFormError(null);
    try {
      const { data, error } = await api.POST('/api/v1/auth/login', {
        body: tenantId ? { ...credentials, tenant_id: tenantId } : credentials,
      });
      if (data) {
        startSession(data);
      } else if (error && 'tenants' in error) {
        setChoice({ credentials, tenants: error.tenants });
      } else {
        setFormError(errorMessage(error, 'No pudimos iniciar sesión. Intente de nuevo.'));
      }
    } catch {
      setFormError(NETWORK_ERROR_MESSAGE);
    }
  }

  async function chooseTenant(tenantId: string) {
    if (!choice) return;
    setPendingTenant(tenantId);
    await login(choice.credentials, tenantId);
    setPendingTenant(null);
  }

  if (choice) {
    return (
      <AuthPage
        title="¿A qué empresa quiere entrar?"
        description="Su correo está registrado en varias empresas. Elija una para continuar."
      >
        {formError && <Alert tone="danger">{formError}</Alert>}
        <div className={styles.tenants}>
          {choice.tenants.map((tenant) => (
            <Button
              key={tenant.id}
              variant="secondary"
              size="lg"
              className={styles.tenant}
              loading={pendingTenant === tenant.id}
              disabled={pendingTenant !== null}
              onClick={() => void chooseTenant(tenant.id)}
            >
              {tenant.name}
            </Button>
          ))}
        </div>
        <Button
          variant="ghost"
          className={styles.back}
          disabled={pendingTenant !== null}
          onClick={() => {
            setChoice(null);
            setFormError(null);
          }}
        >
          Usar otro correo
        </Button>
      </AuthPage>
    );
  }

  return (
    <AuthPage
      title="Ingresar a Cuadra"
      footer={
        <>
          ¿Su negocio aún no usa Cuadra?{' '}
          <AuthLink to={paths.signUp()} state={location.state}>
            Cree una cuenta
          </AuthLink>
        </>
      }
    >
      <form className={styles.form} noValidate onSubmit={handleSubmit((values) => login(values))}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Correo" error={errors.email?.message}>
          <Input type="email" autoComplete="username" inputMode="email" autoFocus {...register('email')} />
        </Field>
        <Field label="Contraseña" error={errors.password?.message}>
          <Input type="password" autoComplete="current-password" {...register('password')} />
        </Field>
        <Button type="submit" size="lg" className={styles.submit} loading={isSubmitting}>
          Ingresar
        </Button>
      </form>
    </AuthPage>
  );
}
