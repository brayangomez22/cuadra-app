import { ApiError, unwrapEmpty } from './errors';

describe('unwrapEmpty', () => {
  it('acepta un 204 sin cuerpo', () => {
    expect(() => {
      unwrapEmpty({ response: new Response(null, { status: 204 }) });
    }).not.toThrow();
  });

  it('lanza ApiError con el código y el mensaje de un error', () => {
    const body = { error: { code: 'unit_in_use', message: 'Hay productos con la unidad Metro.' } };
    const result = { error: body, response: new Response(null, { status: 409 }) };

    expect(() => {
      unwrapEmpty(result);
    }).toThrow(
      expect.objectContaining({
        status: 409,
        code: 'unit_in_use',
        message: 'Hay productos con la unidad Metro.',
      }) as ApiError,
    );
  });
});
