import { paths } from '@/app/routes/paths';
import { AuthLink, AuthPage } from '../../components/AuthPage/AuthPage';
import { SignUpForm } from './components/SignUpForm/SignUpForm';

export function SignUpPage() {
  return (
    <AuthPage
      size="wide"
      title="Cree la cuenta de su negocio"
      description="Usted queda como dueño y después puede agregar a sus empleados."
      footer={
        <>
          ¿Ya tiene cuenta? <AuthLink to={paths.login()}>Ingrese</AuthLink>
        </>
      }
    >
      <SignUpForm />
    </AuthPage>
  );
}
