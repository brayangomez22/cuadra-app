import { useState } from 'react';
import type { Me } from '../../api/types';
import { roleLabels } from '../../model/roles';
import { Button, ButtonLink } from '@/shared/ui';
import { AuthPage } from '../AuthPage/AuthPage';
import styles from './SessionOpenNotice.module.scss';

type Props = {
  user: Me;
  /** Where "Volver a la app" leads. */
  appTarget: string;
  /** What the visitor came to do instead ("Cerrar sesión y elegir otra demo"). */
  switchLabel: string;
  onLogout: () => Promise<void>;
};

// SessionOpenNotice is what the login, the sign-up and the demo show to
// someone who already has a session: who they are, a way back to the app and a
// way out to do what they came for.
export function SessionOpenNotice({ user, appTarget, switchLabel, onLogout }: Props) {
  const [leaving, setLeaving] = useState(false);

  async function logout() {
    setLeaving(true);
    await onLogout();
  }

  return (
    <AuthPage
      title="Ya tiene una sesión abierta"
      description={`${user.name} · ${roleLabels[user.role]} en ${user.tenant_name}.`}
    >
      <div className={styles.root}>
        <ButtonLink to={appTarget}>Volver a la app</ButtonLink>
        <Button variant="secondary" loading={leaving} onClick={logout}>
          {switchLabel}
        </Button>
      </div>
    </AuthPage>
  );
}
