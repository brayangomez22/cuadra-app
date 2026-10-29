// The query keys of the feature. The current user lives under meQueryKey:
// null when there is no session.
export const meQueryKey = ['me'] as const;

export const businessTemplatesQueryKey = ['auth', 'business-templates'] as const;

export const demosQueryKey = ['auth', 'demos'] as const;
