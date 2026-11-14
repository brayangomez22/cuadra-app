import clsx from 'clsx';
import { useState } from 'react';
import { roleLabels, type Me } from '@/features/auth';
import { Button, IconButton, IconLogout } from '@/shared/ui';
import { ThemeSwitch } from './ThemeSwitch/ThemeSwitch';
import styles from './AccountPanel.module.scss';

type Props = {
  user: Me;
  onLogout: () => Promise<void>;
  className?: string;
  /** The folded side menu: only the buttons for the theme and to leave. */
  compact?: boolean;
};

// AccountPanel says who is working and in which company, lets them choose the
// theme of this device, and leave.
export function AccountPanel({ user, onLogout, className, compact = false }: Props) {
  const [leaving, setLeaving] = useState(false);

  async function logout() {
    setLeaving(true);
    await onLogout();
  }

  if (compact) {
    return (
      <div className={clsx(styles.root, styles.compact, className)}>
        <ThemeSwitch compact />
        <IconButton
          label="Cerrar sesión"
          icon={<IconLogout />}
          disabled={leaving}
          aria-busy={leaving}
          onClick={logout}
        />
      </div>
    );
  }

  return (
    <div className={clsx(styles.root, className)}>
      <p className={styles.tenant}>{user.tenant_name}</p>
      <p className={styles.user}>
        <span className={styles.name}>{user.name}</span>
        <span className={styles.role}>{roleLabels[user.role]}</span>
      </p>
      <ThemeSwitch />
      <Button variant="ghost" icon={<IconLogout />} className={styles.logout} loading={leaving} onClick={logout}>
        Cerrar sesión
      </Button>
    </div>
  );
}
