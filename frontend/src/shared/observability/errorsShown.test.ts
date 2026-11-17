import { errorShown, onErrorShown } from './errorsShown';

describe('errorsShown', () => {
  it('avisa el código del error a quien escucha, y deja de avisar al dejar de escuchar', () => {
    const listener = vi.fn();
    const stop = onErrorShown(listener);

    errorShown('insufficient_stock');
    errorShown(undefined);
    stop();
    errorShown('sku_taken');

    expect(listener.mock.calls).toEqual([['insufficient_stock'], ['network_error']]);
  });
});
