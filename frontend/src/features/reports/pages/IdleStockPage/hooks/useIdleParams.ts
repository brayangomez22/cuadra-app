import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';

export const IDLE_PAGE_SIZE = 20;
export const IDLE_DAYS = ['30', '60', '90', '180', '365'] as const;

const daysParam = enumParam<(typeof IDLE_DAYS)[number]>(IDLE_DAYS, '60');
const resetPage = { reset: ['pagina'] };

/** The idle stock report's days, category and page, in the URL (?dias=&categoria=&pagina=). */
export function useIdleParams() {
  const [days, setDays] = useSearchParamsState('dias', daysParam, resetPage);
  const [category, setCategory] = useSearchParamsState('categoria', stringParam(), resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  return {
    days,
    setDays: (next: string) => setDays(daysParam.parse(next) ?? '60'),
    category,
    setCategory,
    offset: (page - 1) * IDLE_PAGE_SIZE,
    setOffset: (next: number) => setPage(next / IDLE_PAGE_SIZE + 1),
  };
}

export type IdleParams = ReturnType<typeof useIdleParams>;
