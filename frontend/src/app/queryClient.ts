import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      // authFetch already retries once after renewing the session; on top of that,
      // a single retry for transient failures is enough.
      retry: 1,
    },
  },
});
