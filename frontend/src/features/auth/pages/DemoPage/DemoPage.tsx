import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { ButtonLink, EmptyState } from '@/shared/ui';
import { useDemos } from '../../api/demo';
import { AuthLink, AuthPage } from '../../components/AuthPage/AuthPage';
import { DemoForm } from './components/DemoForm/DemoForm';

/** /demo: anyone tries Cuadra on an example business, without an account. */
export function DemoPage() {
  const demos = useDemos();

  return (
    <AuthPage
      title="Probar la demo"
      description="Entre a un negocio de ejemplo con seis meses de ventas, compras e inventario. Puede cambiar lo que quiera: los datos se reinician cada noche."
      footer={
        <>
          ¿Ya tiene cuenta? <AuthLink to={paths.login()}>Ingrese</AuthLink>
        </>
      }
    >
      <QueryState
        queries={[demos]}
        errorTitle="No pudimos cargar la demostración"
        isEmpty={demos.data?.length === 0}
        empty={
          <EmptyState
            title="La demostración no está disponible"
            description="Intente más tarde, o ingrese con su cuenta."
            action={<ButtonLink to={paths.login()}>Ingresar</ButtonLink>}
          />
        }
      >
        {() => <DemoForm demos={demos.data ?? []} />}
      </QueryState>
    </AuthPage>
  );
}
