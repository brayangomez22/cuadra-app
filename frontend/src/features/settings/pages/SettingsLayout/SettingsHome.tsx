import { Navigate } from 'react-router';
import { useAccess, visibleItems } from '@/features/auth';
import { sections } from '../../model/sections';

/** /configuracion opens the first section the user may see. */
export function SettingsHome() {
  const [first] = visibleItems(sections, useAccess());
  return first ? <Navigate to={first.to} replace /> : null;
}
