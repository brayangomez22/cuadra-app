# Convención de estilos (SCSS + CSS Modules)

Objetivo: una UI coherente, accesible y fácil de mantener, construida por nosotros y sin un framework de utilidades. Cualquier valor visual sale de un **token**. Cada componente encapsula sus estilos en su propio módulo.

## 1. Herramientas

- `sass-embedded` (Vite compila `.scss` de forma nativa una vez instalado).
- **CSS Modules**: cada `*.module.scss` tiene sus clases con alcance local automático. Dos componentes no pueden chocar aunque usen el mismo nombre de clase.
- **Nombres legibles en las DevTools** (configuración en `vite.config.ts`):
  ```ts
  css: {
    modules: {
      localsConvention: 'camelCaseOnly',
      generateScopedName: process.env.NODE_ENV === 'production'
        ? '[hash:base64:6]'
        : '[name]__[local]__[hash:base64:4]',
    },
  },
  ```
  En desarrollo verás `Button-module__icon__x7f2`; en producción, clases cortas.
- `clsx` para componer clases condicionales.
- **Radix Primitives** (sin estilos) para piezas difíciles de hacer accesibles: Dialog, DropdownMenu, Popover, Tooltip, Tabs, Select y Toast. Les ponemos nuestro SCSS encima.
- **stylelint** con:
  - `stylelint-config-standard-scss`,
  - `selector-class-pattern: '^[a-z][a-zA-Z0-9]*$'` (camelCase),
  - `stylelint-declaration-strict-value`, que obliga a usar tokens para colores, z-index y sombras.
- Tipado: `src/vite-env.d.ts` declara los `*.module.scss`. Opcionalmente, `typescript-plugin-css-modules` da autocompletado de clases en el editor.

## 2. Estructura

```
src/styles/
  tokens/
    _primitives.scss     # paleta cruda y escalas (mapas SCSS). NUNCA se usa desde componentes
    _semantic.scss       # tokens con significado, emitidos como CSS custom properties
    _typography.scss     # familias, escala tipográfica, pesos, interlineados
    _spacing.scss        # escala de espacio (base 4px)
    _radius.scss  _shadow.scss  _motion.scss  _z-index.scss
    _breakpoints.scss    # mapa de breakpoints (solo para el mixin)
  abstracts/
    _mixins.scss         # respond-to, focus-ring, visually-hidden, truncate, touch-target, reduced-motion-safe
    _functions.scss      # rem(), etc.
    _index.scss          # @forward de mixins y funciones (lo que importan los módulos)
  base/
    _reset.scss          # reset moderno mínimo
    _global.scss         # body, tipografía base, selección, scrollbars
    _a11y.scss           # :focus-visible global, .sr-only
  themes/
    _light.scss          # :root { --color-...: ... }
    _dark.scss           # [data-theme="dark"] { ... } y prefers-color-scheme
  main.scss              # punto de entrada global, importado UNA vez en main.tsx

src/shared/ui/Button/
  Button.tsx
  Button.module.scss
  Button.test.tsx
  index.ts
```

**Regla de oro:** los estilos globales (reset, tipografía base, temas, utilidades de accesibilidad) viven **solo** en `src/styles/`. Todo lo demás es un módulo.

## 3. Tokens: dos capas

**Capa 1, primitivos** (SCSS, privados): la materia prima.
```scss
// tokens/_primitives.scss
$gray: (50: #f7f7f5, 100: #eeede9, /* ... */ 900: #1c1b19);
$brand: (500: #..., 600: #...);
```

**Capa 2, semánticos** (CSS custom properties, públicos): describen el **uso**, no el color. Son lo único que tocan los componentes y lo que cambia entre temas.
```scss
// themes/_light.scss
@use 'sass:map';
@use '../tokens/primitives' as p;
:root {
  --color-bg-canvas:      #{map.get(p.$gray, 50)};
  --color-bg-surface:     #fff;
  --color-bg-subtle:      #{map.get(p.$gray, 100)};
  --color-text-default:   #{map.get(p.$gray, 900)};
  --color-text-muted:     #{map.get(p.$gray, 600)};
  --color-border-default: #{map.get(p.$gray, 200)};
  --color-action-primary: #{map.get(p.$brand, 600)};
  --color-action-primary-hover: #{map.get(p.$brand, 700)};
  --color-text-on-action: #fff;
  --color-feedback-danger:  ...;   // también success, warning, info
  --color-focus-ring: ...;
}
```

Nombres de tokens: `--{categoría}-{rol}-{variante}-{estado}`. Por ejemplo, `--color-action-danger-hover`.

**Escalas** (también como custom properties):
- Espacio: `--space-1` (4px) `--space-2` (8) `--space-3` (12) `--space-4` (16) `--space-5` (20) `--space-6` (24) `--space-8` (32) `--space-10` (40) `--space-12` (48) `--space-16` (64).
- Tipografía: `--font-size-xs … --font-size-3xl`, `--line-height-tight|normal`, `--font-weight-regular|medium|semibold|bold`, `--font-family-sans|mono`. Usa `font-variant-numeric: tabular-nums` para cifras, precios y cantidades en tablas.
- Radios: `--radius-sm|md|lg|full`. Sombras: `--shadow-sm|md|lg`. Movimiento: `--duration-fast|normal`, `--ease-standard`. Capas: `--z-dropdown|sticky|modal|toast`.

## 4. Cómo se escribe un componente

```scss
// shared/ui/Button/Button.module.scss
@use '@/styles/abstracts' as *;

.root {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font-weight: var(--font-weight-medium);
  background: var(--color-action-primary);
  color: var(--color-text-on-action);
  transition: background-color var(--duration-fast) var(--ease-standard);
  @include touch-target;
  @include focus-ring;

  &:hover:not(:disabled) { background: var(--color-action-primary-hover); }
  &:disabled { opacity: 0.5; cursor: not-allowed; }
  &[aria-busy='true'] { cursor: progress; }

  &[data-variant='secondary'] {
    background: var(--color-bg-surface);
    border-color: var(--color-border-default);
    color: var(--color-text-default);
  }
  &[data-variant='danger'] { background: var(--color-action-danger); }
  &[data-size='lg'] { padding: var(--space-3) var(--space-6); font-size: var(--font-size-lg); }
}

.icon { flex-shrink: 0; }
.label { @include truncate; }
```

```tsx
// shared/ui/Button/Button.tsx
import clsx from 'clsx';
import styles from './Button.module.scss';

type Props = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'md' | 'lg';
  loading?: boolean;
};

export function Button({ variant = 'primary', size = 'md', loading, className, children, ...rest }: Props) {
  return (
    <button
      className={clsx(styles.root, className)}
      data-variant={variant}
      data-size={size}
      aria-busy={loading || undefined}
      {...rest}
    >
      <span className={styles.label}>{children}</span>
    </button>
  );
}
```

Todo componente acepta `className` y lo combina con su `.root`. Así el padre puede **posicionarlo** (márgenes, `grid-area`, ancho) desde su propio módulo, sin tocar cómo se ve:
```tsx
<Button className={styles.checkout}>Cobrar</Button>   // .checkout vive en PosCart.module.scss
```

## 5. Reglas

**Nombres de clase**
- En camelCase dentro del módulo (`styles.rowActions`). La clase raíz del componente se llama siempre `.root`.
- Nombra por **qué es**, no por cómo se ve: `.priceCell` sí, `.boldRight` no.
- Usa nombres cortos y locales (`.header`, `.total`, `.line`). El módulo ya les da contexto; no repitas el nombre del componente (`.posCartTotal` sobra).

**Variantes y estados**
- Las variantes van en `data-*` (`data-variant`, `data-size`, `data-tone`, `data-density`). Así no se multiplican clases en el TSX y el valor se ve en el HTML.
- Los estados usan atributos nativos o ARIA: `:disabled`, `[aria-invalid='true']`, `[aria-expanded='true']`, `[aria-selected='true']`, `[aria-current='page']`, `[aria-busy='true']`, `[data-state='open']` (el que pone Radix). Así el estilo y la accesibilidad no se pueden desincronizar.
- Para un estado sin equivalente ARIA, usa `data-state='...'`.

**Valores**
- En los módulos solo `var(--token)`. Nada de hex, `rgb()` o `px` arbitrarios. Se permiten `0`, `1px` en bordes, `%`, `fr` y `ch`.
- Si necesitas un valor que no existe, **agrega un token** en `styles/tokens` (y documéntalo). No lo inventes en el componente.
- Los primitivos (`$gray`, `$brand`) nunca se usan fuera de `styles/`.

**Estructura del SCSS**
- Usa siempre `@use` / `@forward`; `@import` está deprecado en Sass.
- Anidación máxima de 3 niveles. Úsala para pseudo-clases, estados y variantes, no para replicar el DOM (`.root .header .title` está mal; `.title` basta).
- Sin `!important`, sin IDs y sin selectores de etiqueta (excepto dentro de `.root` para contenido de terceros, por ejemplo HTML enriquecido).
- `:global(...)` solo con un comentario que explique por qué.
- No se usa `composes`; para compartir estilos usa mixins de `abstracts` o un componente.
- Orden dentro de un bloque: posición y layout → caja (tamaño, padding, borde) → tipografía → colores y fondo → efectos (sombra, transición) → `@include` → estados → variantes → anidados.
- Un componente no define su propio `margin` externo. El espacio entre componentes lo pone el padre con `gap` o pasando `className`.

**Layout y responsive**
- Flex o grid con `gap`.
- Mobile-first: `@include respond-to(md) { ... }`. Breakpoints `sm` 640, `md` 768, `lg` 1024, `xl` 1280.
- Prefiere propiedades lógicas (`margin-inline`, `padding-block`).
- Tablas anchas dentro de un contenedor con `overflow-x: auto`.

**Accesibilidad (no negociable)**
- Contraste de texto 4.5:1 (3:1 para textos de 24px o más y para íconos). Verifícalo al definir los tokens, no después.
- `@include focus-ring` en todo lo interactivo. Nunca `outline: none` sin reemplazo.
- `@include touch-target` (44×44px mínimo) en botones y controles del POS.
- No transmitas información solo con color: un estado de error lleva ícono y texto.
- Las animaciones se envuelven en `@include reduced-motion-safe`.

**Temas**
- Claro por defecto; oscuro con `[data-theme='dark']` sobre `<html>` y fallback con `prefers-color-scheme`.
- Como los componentes solo usan tokens semánticos, un tema nuevo no exige tocar ningún componente.

## 6. Particularidades de este producto

- **El POS es una herramienta de trabajo**, no una landing: prima la velocidad, la densidad legible y el teclado (lector de código de barras, atajos, Enter para confirmar). Las decoraciones son mínimas.
- **Las cifras mandan**: precios y cantidades alineados a la derecha, con números tabulares y separador de miles colombiano (`$ 12.500`).
- Usuarios de 35 a 60 años, a veces en mostradores con mala luz y pantallas modestas. El tamaño base de texto debe ser cómodo (16px como mínimo) y con alto contraste.
- Habrá tablets y celulares, así que los objetivos táctiles importan.

## 7. Checklist antes de dar por terminado un componente

- [ ] Un `.module.scss` propio, con `.root` y sin estilos globales.
- [ ] Solo tokens semánticos y `npm run lint:styles` en verde.
- [ ] Estados hover, focus-visible, active, disabled y, si aplica, loading, error y vacío.
- [ ] Funciona con teclado y con lector de pantalla (rol, label y `aria-*`).
- [ ] Se ve bien en 360px y en 1440px, en tema claro y oscuro.
- [ ] Tiene test de comportamiento (Testing Library) y, si es visual, revisión con captura de pantalla.
