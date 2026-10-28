import { Link, useLocation } from 'react-router';
import { visibleItems, type Access } from '@/features/auth';
import { navItems, type NavItem } from './navItems';
import styles from './MainNav.module.scss';

type Props = {
  access: Access;
  /** Called when an item is chosen, e.g. to close the menu panel on a phone. */
  onNavigate?: () => void;
};

export function MainNav({ access, onNavigate }: Props) {
  const items = visibleItems(navItems, access);
  const current = currentItem(items, useLocation().pathname);
  return (
    <nav aria-label="Menú principal" className={styles.root}>
      <ul className={styles.list}>
        {items.map(({ to, label, icon: Icon }) => (
          <li key={to}>
            <Link
              to={to}
              className={styles.link}
              aria-current={to === current ? 'page' : undefined}
              onClick={onNavigate}
            >
              <Icon className={styles.icon} />
              <span className={styles.label}>{label}</span>
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}

// currentItem is the item of the page: the one whose URL is the longest
// prefix of the path, so a page under another item's URL (Toma física under
// Inventario) marks only its own. Inicio marks only its exact URL.
function currentItem(items: NavItem[], pathname: string) {
  const matches = items.filter(({ to }) => pathname === to || (to !== '/' && pathname.startsWith(`${to}/`)));
  return matches.sort((a, b) => b.to.length - a.to.length)[0]?.to;
}
