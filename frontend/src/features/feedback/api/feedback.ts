import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

type FeedbackInput = components['schemas']['FeedbackInput'];

/** Sends a comment; nothing is cached, the team reads them. */
export function useSendFeedback() {
  return useMutation({
    mutationFn: async (body: FeedbackInput) => unwrap(await api.POST('/api/v1/feedback', { body })),
  });
}
