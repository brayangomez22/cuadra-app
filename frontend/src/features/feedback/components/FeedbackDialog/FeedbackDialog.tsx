import { zodResolver } from '@hookform/resolvers/zod';
import { useId, type ReactElement } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation } from 'react-router';
import { track } from '@/features/analytics';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { Alert, Button, Dialog, DialogClose, Field, Stack, Textarea, useToast } from '@/shared/ui';
import { useSendFeedback } from '../../api/feedback';
import { feedbackSchema } from '../../model/feedbackSchema';

type Props = { open: boolean; onOpenChange: (open: boolean) => void; trigger: ReactElement };

/** The comment's form. Of the screen only the path goes: the query may carry what the user searched. */
export function FeedbackDialog({ open, onOpenChange, trigger }: Props) {
  const formId = useId();
  const { pathname } = useLocation();
  const { toast } = useToast();
  const form = useForm({ resolver: zodResolver(feedbackSchema), defaultValues: { message: '' } });
  const { mutateAsync } = useSendFeedback();
  const { formError, handleError } = useApiFormErrors(form.setError, { invalid_feedback: 'message' });
  const submit = form.handleSubmit(async ({ message }) => {
    try {
      await mutateAsync({ message, screen: pathname, app_version: import.meta.env.VITE_RELEASE ?? 'dev' });
      void track('feedback_sent', { screen: pathname });
      toast({ tone: 'success', title: '¡Gracias por su comentario!' });
      form.reset();
      onOpenChange(false);
    } catch (error) {
      handleError(error);
    }
  });
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      trigger={trigger}
      title="Danos tu opinión"
      description="Cuéntenos qué le sirve, qué le falta o qué no funciona. Lo lee el equipo de Cuadra."
      size="md"
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={form.formState.isSubmitting}>
            Enviar
          </Button>
        </>
      }
    >
      <form id={formId} noValidate onSubmit={(event) => void submit(event)}>
        <Stack>
          {formError && <Alert tone="danger">{formError}</Alert>}
          <Field label="Su comentario" error={form.formState.errors.message?.message}>
            <Textarea rows={5} maxLength={2000} {...form.register('message')} />
          </Field>
        </Stack>
      </form>
    </Dialog>
  );
}
