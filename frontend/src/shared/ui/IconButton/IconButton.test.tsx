import { render, screen } from '@testing-library/react';
import { IconClose } from '../icons';
import { IconButton } from './IconButton';

describe('IconButton', () => {
  it('expone su label como nombre accesible', () => {
    render(<IconButton label="Quitar producto" icon={<IconClose />} />);

    expect(screen.getByRole('button', { name: 'Quitar producto' })).toBeInTheDocument();
  });
});
