import type { FeatureFlag } from '../api/types';

export const pilotId = '01920000-0000-7000-8000-0000000000aa';
export const clothingId = '01920000-0000-7000-8000-0000000000bb';

/** The flags as an operator sees them: feedback released to a hardware store only. */
export function someFlags(): FeatureFlag[] {
  return [
    {
      key: 'feedback',
      description: 'Muestra el botón «Danos tu opinión» en la app.',
      enabled: false,
      exceptions: [{ tenant_id: pilotId, tenant_name: 'Ferretería La Tuerca', enabled: true }],
    },
    { key: 'analytics', description: 'Envía la analítica de uso de la app.', enabled: true, exceptions: [] },
  ];
}
