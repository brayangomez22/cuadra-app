import { useState } from 'react';
import { BarcodeScanner, Button, Stack } from '@/shared/ui';
import { Section } from '../Section/Section';

/** The scanner opens the camera only when asked, so the catalog does not ask for permission on load. */
export function ScannerSection() {
  const [open, setOpen] = useState(false);
  const [codes, setCodes] = useState<string[]>([]);
  return (
    <Section title="BarcodeScanner">
      <Stack gap="3">
        <Button variant="secondary" onClick={() => setOpen(!open)}>
          {open ? 'Cerrar la cámara' : 'Abrir la cámara'}
        </Button>
        {open && <BarcodeScanner onScan={(code) => setCodes((read) => [code, ...read].slice(0, 5))} />}
        {codes.length > 0 && <p>Leídos: {codes.join(', ')}</p>}
      </Stack>
    </Section>
  );
}
