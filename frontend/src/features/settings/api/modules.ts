import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { meQueryKey } from '@/features/auth';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { settingsKeys } from './keys';

export function useModules() {
  return useQuery({
    queryKey: settingsKeys.modules,
    queryFn: async () => unwrap(await api.GET('/api/v1/modules')).items,
  });
}

/** Turns a module on or off. /me lists the active modules: the menus and fields follow it. */
export function useSetModuleActive() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) =>
      unwrap(await api.PUT('/api/v1/modules/{moduleId}', { params: { path: { moduleId: id } }, body: { active } })),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: settingsKeys.modules }),
        queryClient.invalidateQueries({ queryKey: meQueryKey }),
      ]),
  });
}
