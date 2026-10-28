import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission, type Gated } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';
import { SectionGate } from './components/SectionGate/SectionGate';

// gate guards a group of sections inside the settings layout, which keeps the
// page's title: the section answers "no access" in its place.
function gate(access: Gated): Pick<RouteObject, 'element'> {
  return { element: <SectionGate {...access} /> };
}

// screen loads one of the settings screens of pages.ts, in the feature's chunk.
const screen = (name: keyof typeof import('./pages')) => lazyPage(async () => (await import('./pages'))[name]);

/**
 * Settings: owners and admins (catalog:write); each group of sections asks
 * for its own permission or module.
 */
export const settingsRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="catalog:write" />,
    children: [
      {
        path: paths.settings.home.pattern,
        ...screen('SettingsLayout'),
        children: [
          { index: true, ...screen('SettingsHome') },
          { path: paths.settings.units.pattern, ...screen('UnitsSettings') },
          { path: paths.settings.attributes.pattern, ...screen('AttributesSettings') },
          {
            ...gate({ module: 'variants' }),
            children: [{ path: paths.settings.variantAxes.pattern, ...screen('VariantAxesSettings') }],
          },
          {
            ...gate({ permission: 'customers:manage' }),
            children: [{ path: paths.settings.customerSegments.pattern, ...screen('CustomerSegmentsSettings') }],
          },
          {
            ...gate({ permission: 'tenant:manage' }),
            children: [
              { path: paths.settings.general.pattern, ...screen('GeneralSettings') },
              { path: paths.settings.modules.pattern, ...screen('ModulesSettings') },
            ],
          },
        ],
      },
    ],
  },
];
