import { paths } from '@/app/routes/paths';
import { describeError } from '@/shared/api/errors';
import { Alert } from '@/shared/ui';
import { AuthLink, AuthPage } from '../AuthPage/AuthPage';

type Props = {
  title: string;
  /** The failed request, or null when the link came without its token. */
  error: unknown;
};

const incomplete = 'El enlace está incompleto. Ábralo tal como se lo enviaron o pida uno nuevo.';

/** A link that cannot be used (incomplete, expired, used): what happened and how to go on. */
export function LinkProblem({ title, error }: Props) {
  return (
    <AuthPage
      title={title}
      footer={
        <>
          ¿Ya tiene cuenta? <AuthLink to={paths.login()}>Ingrese</AuthLink>
        </>
      }
    >
      <Alert tone="warning">{error === null ? incomplete : describeError(error).message}</Alert>
    </AuthPage>
  );
}
