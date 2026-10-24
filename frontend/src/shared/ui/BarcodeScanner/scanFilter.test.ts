import { createScanFilter } from './scanFilter';

const QUIET = 1500;

// The frames the camera sees over time: [ms, codes in view].
type Frame = [number, string[]];

function fired(frames: Frame[]) {
  const accept = createScanFilter(QUIET);
  return frames.map(([now, codes]) => accept(codes, now)).filter((code) => code !== undefined);
}

describe('createScanFilter', () => {
  it.each<{ name: string; frames: Frame[]; expected: string[] }>([
    {
      name: 'dispara una sola vez aunque el código siga en cámara (EAN-13 de una droguería)',
      frames: [
        [0, ['7702057000123']],
        [150, ['7702057000123']],
        [300, ['7702057000123']],
        [1600, ['7702057000123']],
      ],
      expected: ['7702057000123'],
    },
    {
      name: 'vuelve a disparar el mismo código cuando dejó de verse (Code 128 de un almacén de ropa)',
      frames: [
        [0, ['CAM-AZ-M']],
        [150, []],
        [1700, ['CAM-AZ-M']],
      ],
      expected: ['CAM-AZ-M', 'CAM-AZ-M'],
    },
    {
      name: 'no vuelve a disparar si el código desaparece menos que el tiempo de espera',
      frames: [
        [0, ['7701234567890']],
        [150, []],
        [900, ['7701234567890']],
      ],
      expected: ['7701234567890'],
    },
    {
      name: 'un código distinto dispara de inmediato',
      frames: [
        [0, ['7701234567890']],
        [150, ['CAM-AZ-M']],
      ],
      expected: ['7701234567890', 'CAM-AZ-M'],
    },
    {
      name: 'con dos códigos a la vez dispara solo el primero',
      frames: [
        [0, ['7702057000123', 'LOTE-88']],
        [150, ['7702057000123', 'LOTE-88']],
      ],
      expected: ['7702057000123'],
    },
    { name: 'sin códigos no dispara', frames: [[0, []]], expected: [] },
  ])('$name', ({ frames, expected }) => {
    expect(fired(frames)).toEqual(expected);
  });
});
