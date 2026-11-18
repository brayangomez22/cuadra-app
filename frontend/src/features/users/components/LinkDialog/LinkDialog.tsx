import { Button, DialogClose, Dialog, Field, IconCheck, Input, Stack } from '@/shared/ui';
import { formatDateTime } from '@/shared/format';
import { useCopy } from '../../hooks/useCopy';

/** A link to share once it is issued: its address and when it expires. */
export type IssuedLinkView = { title: string; description: string; url: string; expiresAt: string };

type Props = { link: IssuedLinkView | null; onClose: () => void };

/**
 * Shows the link of an invitation or a password reset, to copy and send to
 * the person (by WhatsApp, a message…): Cuadra does not email it yet. It is
 * shown only now; issuing another one replaces it.
 */
export function LinkDialog({ link, onClose }: Props) {
  const { copied, copy } = useCopy();
  return (
    <Dialog
      open={link !== null}
      onOpenChange={(open) => !open && onClose()}
      title={link?.title ?? ''}
      description={link?.description ?? ''}
      size="sm"
      footer={
        <DialogClose asChild>
          <Button variant="secondary">Listo</Button>
        </DialogClose>
      }
    >
      {link && (
        <Stack gap="3">
          <Field label="Enlace" hint={`Sirve una sola vez. Vence el ${formatDateTime(link.expiresAt)}.`}>
            <Input readOnly value={link.url} onFocus={(event) => event.currentTarget.select()} />
          </Field>
          <Button icon={copied ? <IconCheck /> : undefined} onClick={() => void copy(link.url)}>
            {copied ? 'Enlace copiado' : 'Copiar enlace'}
          </Button>
        </Stack>
      )}
    </Dialog>
  );
}
