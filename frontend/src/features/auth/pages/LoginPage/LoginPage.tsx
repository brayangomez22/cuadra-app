import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { ButtonLink } from '@/shared/ui';
import { useDemos } from '../../api/demo';
import { AuthLink, AuthPage } from '../../components/AuthPage/AuthPage';
import { loginSchema, type LoginValues } from '../../model/schemas';
import { LoginForm } from './components/LoginForm/LoginForm';
import { TenantChoice } from './components/TenantChoice/TenantChoice';
import { useLogin } from './hooks/useLogin';

export function LoginPage() {
  const form = useForm<LoginValues>({ resolver: zodResolver(loginSchema), defaultValues: { email: '', password: '' } });
  const { formError, choice, pendingTenant, submit, chooseTenant, back } = useLogin(form.setError);
  const hasDemo = (useDemos().data?.length ?? 0) > 0;

  if (choice) {
    return (
      <TenantChoice
        tenants={choice.tenants}
        formError={formError}
        pendingTenant={pendingTenant}
        onChoose={(tenantId) => void chooseTenant(tenantId)}
        onBack={back}
      />
    );
  }

  return (
    <AuthPage
      title="Ingresar a Cuadra"
      footer={
        <>
          ¿Su negocio aún no usa Cuadra? <AuthLink to={paths.signUp()}>Cree una cuenta</AuthLink>
        </>
      }
    >
      <LoginForm form={form} formError={formError} onSubmit={(values) => submit(values)} />
      {hasDemo && (
        <ButtonLink variant="secondary" size="lg" to={paths.demo()}>
          Probar la demo
        </ButtonLink>
      )}
    </AuthPage>
  );
}
