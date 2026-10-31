import { Field, Select } from '@/shared/ui';
import type { Paper } from '../../model/paper';

const options = [
  { value: '80', label: '80 mm' },
  { value: '58', label: '58 mm' },
];

type Props = { paper: Paper; onPaperChange: (paper: Paper) => void; className?: string };

/** The width of the roll the receipt prints on. */
export function PaperSelect({ paper, onPaperChange, className }: Props) {
  return (
    <Field label="Papel" className={className}>
      <Select options={options} value={paper} onValueChange={(value) => onPaperChange(value === '58' ? '58' : '80')} />
    </Field>
  );
}
