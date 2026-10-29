import clsx from 'clsx';
import { useState } from 'react';
import { roleLabels, type Me } from '@/features/auth';
import { Button, IconLogout } from '@/shared/ui';
import styles from './AccountPanel.module.scss';

type Props = {
  user: Me;
  onLogout: () => Promise<void>;
  className?: string;
};

// AccountPanel says who is working and in which company, and lets them leave.
export function AccountPanel({ user, onLogout, className }: Props) {
  const [leaving, setLeaving] = useState(false);

  async function logout() {
    setLeaving(true);
    await onLogout();
  }

  return (
    <div className={clsx(styles.root, className)}>
      <p className={styles.tenant}>{user.tenant_name}</p>
      <p className={styles.user}>
        <span className={styles.name}>{user.name}</span>
        <span className={styles.role}>{roleLabels[user.role]}</span>
      </p>
      <Button variant="ghost" icon={<IconLogout />} className={styles.logout} loading={leaving} onClick={logout}>
        Cerrar sesión
      </Button>
    </div>
  );
}
