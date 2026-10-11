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
    _sizes.scss          # área táctil mínima y densidad (alto de controles y filas)
    _breakpoints.scss    # mapa de breakpoints (solo para el mixin)
  abstracts/
    _mixins.scss         # respond-to, hover, focus-ring, visually-hidden, truncate, touch-target, numeric, reduced-motion-safe
    _functions.scss      # rem(), etc.
    _index.scss          # @forward de mixins y funciones (lo que importan los módulos)
  base/
    _tokens.scss         # escalas que no cambian con el tema, como custom properties en :root
    _reset.scss          # reset moderno mínimo
    _global.scss         # body, tipografía base, selección
    _a11y.scss           # :focus-visible global, .srOnly
  themes/
    _theme.scss          # mixin que emite los colores y sombras de un tema
    _light.scss          # :root { --color-...: ... }
    _dark.scss           # [data-theme="dark"] { ... } y prefers-color-scheme
  main.scss              # punto de entrada global, importado UNA vez en main.tsx

frontend/styles-check/   # tests de los tokens: contraste, paridad de temas, escalas y mixins

src/shared/ui/Button/
  Button.tsx
  Button.module.scss
  Button.test.tsx
  index.ts
```

**Regla de oro:** los estilos globales (reset, tipografía base, temas, utilidades de accesibilidad) viven **solo** en `src/styles/`. Todo lo demás es un módulo.

## 3. Tokens: dos capas

### Dirección visual: "Señal de obra"

Elegida en T13 entre tres propuestas (ver `docs/decisiones.md`). El naranja de seguridad de la obra es el **único** color de marca, sobre grises cálidos de cemento, con texto casi negro. Las cifras son grandes y el total de la venta usa `--font-size-display` (56px). Densidad cómoda: filas de 48px con mouse y de 56px en pantallas táctiles. Solo fuentes del sistema.

**Reglas del naranja:**
- El naranja es **fondo** (la acción principal) o **acento** (`--color-border-accent`: subrayado de enlaces, fila seleccionada, marca). **Nunca es color de texto:** sobre blanco da 3.6:1.
- Sobre naranja el texto siempre es oscuro (`--color-text-on-action`). Por eso el hover del botón primario se **aclara** y el active se oscurece, sin bajar de 4.5:1.
- Un enlace se escribe en `--color-text-default` con subrayado en `--color-border-accent`.
- Una pantalla tiene un solo elemento naranja dominante: la acción principal (por ejemplo, "Cobrar").

**Capa 1, primitivos** (SCSS, privados): la materia prima. Viven en `tokens/_primitives.scss` (`$neutral`, `$orange`, `$red`, `$green`, `$amber`, `$blue`) y nunca se usan fuera de `styles/`.

**Capa 2, semánticos** (CSS custom properties, públicos): describen el **uso**, no el color. Son lo único que tocan los componentes y lo que cambia entre temas. Se definen una vez por tema como mapas en `tokens/_semantic.scss` (`$light`, `$dark`), y `themes/` los emite como `--color-<rol>`:

| Grupo | Tokens |
|---|---|
| Fondos | `--color-bg-canvas` (fondo de la app), `--color-bg-surface` (tarjetas, tablas, inputs), `--color-bg-subtle` (encabezados de tabla, fila seleccionada) |
| Texto | `--color-text-default`, `--color-text-muted`, `--color-text-on-action` (sobre naranja), `--color-text-on-danger` |
| Bordes | `--color-border-default` (separadores, decorativo), `--color-border-strong` (borde de inputs y controles: 3:1), `--color-border-accent` (naranja) |
| Acciones | `--color-action-primary` + `-hover` + `-active`, `--color-action-danger` + `-hover` |
| Estados | `--color-feedback-{danger,success,warning,info}` (texto e íconos) y `--color-feedback-{...}-subtle` (su fondo) |
| Otros | `--color-focus-ring`, `--color-overlay` (fondo de los diálogos) |

Nombres de tokens: `--{categoría}-{rol}-{variante}-{estado}`. Por ejemplo, `--color-action-danger-hover`.

**Escalas** (también como custom properties, en `base/_tokens.scss`):
- Espacio: `--space-1` (4px) `--space-2` (8) `--space-3` (12) `--space-4` (16) `--space-5` (20) `--space-6` (24) `--space-8` (32) `--space-10` (40) `--space-12` (48) `--space-16` (64).
- Tipografía: `--font-size-xs` (13px) `sm` (14) `md` (16, el texto base) `lg` (18) `xl` (20) `2xl` (24) `3xl` (32) `display` (56, el total de la venta). `--line-height-tight|normal`, `--font-weight-regular|medium|semibold|bold|heavy`, `--font-family-sans|mono` (fuentes del sistema). Usa `@include numeric` (`tabular-nums`) para cifras, precios y cantidades.
- Tamaños: `--size-control` (alto de botones e inputs: 44px con mouse, 56px táctil), `--size-row` (fila de tabla: 48px con mouse, 56px táctil), `--size-touch-min` (44px). El cambio a táctil lo hace `@media (pointer: coarse)`, sin código en los componentes. Anchos máximos: `--size-dialog-sm` (440px), `--size-dialog-md` (640px), `--size-toast` (400px) y `--size-page` (1200px, el contenido de una página en pantallas anchas); en un celular se reducen para caber.
- Radios: `--radius-sm|md|lg|full`. Sombras: `--shadow-sm|md|lg` (cambian con el tema). Movimiento: `--duration-fast|normal`, `--ease-standard`. Capas: `--z-dropdown|sticky|modal|toast`.

### Verificación de contraste

`frontend/styles-check/tokens.test.ts` compila `main.scss` y comprueba cada par primer plano/fondo permitido en los dos temas: 4.5:1 para texto (WCAG 1.4.3) y 3:1 para el anillo de foco y los bordes de controles (WCAG 1.4.11). Un token de color nuevo agrega sus pares a la lista del test antes de usarse en un componente.

La tabla siguiente la genera ese mismo test; si cambias un color, el test falla y muestra la tabla nueva para pegarla aquí. Los valores están truncados, nunca redondeados hacia arriba.

| Tema | Primer plano | Fondo | Contraste | Mínimo |
|---|---|---|---|---|
| claro | `--color-text-default` | `--color-bg-canvas` | 14.45:1 | 4.5:1 |
| claro | `--color-text-default` | `--color-bg-surface` | 17.56:1 | 4.5:1 |
| claro | `--color-text-default` | `--color-bg-subtle` | 13.17:1 | 4.5:1 |
| claro | `--color-text-muted` | `--color-bg-canvas` | 6.40:1 | 4.5:1 |
| claro | `--color-text-muted` | `--color-bg-surface` | 7.78:1 | 4.5:1 |
| claro | `--color-text-muted` | `--color-bg-subtle` | 5.83:1 | 4.5:1 |
| claro | `--color-text-on-action` | `--color-action-primary` | 5.67:1 | 4.5:1 |
| claro | `--color-text-on-action` | `--color-action-primary-hover` | 6.75:1 | 4.5:1 |
| claro | `--color-text-on-action` | `--color-action-primary-active` | 4.88:1 | 4.5:1 |
| claro | `--color-text-on-danger` | `--color-action-danger` | 6.53:1 | 4.5:1 |
| claro | `--color-text-on-danger` | `--color-action-danger-hover` | 8.12:1 | 4.5:1 |
| claro | `--color-feedback-danger` | `--color-bg-surface` | 6.53:1 | 4.5:1 |
| claro | `--color-feedback-danger` | `--color-feedback-danger-subtle` | 5.57:1 | 4.5:1 |
| claro | `--color-feedback-success` | `--color-bg-surface` | 6.53:1 | 4.5:1 |
| claro | `--color-feedback-success` | `--color-feedback-success-subtle` | 5.62:1 | 4.5:1 |
| claro | `--color-feedback-warning` | `--color-bg-surface` | 6.32:1 | 4.5:1 |
| claro | `--color-feedback-warning` | `--color-feedback-warning-subtle` | 5.56:1 | 4.5:1 |
| claro | `--color-feedback-info` | `--color-bg-surface` | 7.09:1 | 4.5:1 |
| claro | `--color-feedback-info` | `--color-feedback-info-subtle` | 5.98:1 | 4.5:1 |
| claro | `--color-focus-ring` | `--color-bg-canvas` | 14.45:1 | 3:1 |
| claro | `--color-focus-ring` | `--color-bg-surface` | 17.56:1 | 3:1 |
| claro | `--color-border-strong` | `--color-bg-canvas` | 3.67:1 | 3:1 |
| claro | `--color-border-strong` | `--color-bg-surface` | 4.47:1 | 3:1 |
| claro | `--color-border-accent` | `--color-bg-surface` | 3.59:1 | 3:1 |
| oscuro | `--color-text-default` | `--color-bg-canvas` | 15.87:1 | 4.5:1 |
| oscuro | `--color-text-default` | `--color-bg-surface` | 14.33:1 | 4.5:1 |
| oscuro | `--color-text-default` | `--color-bg-subtle` | 12.48:1 | 4.5:1 |
| oscuro | `--color-text-muted` | `--color-bg-canvas` | 8.52:1 | 4.5:1 |
| oscuro | `--color-text-muted` | `--color-bg-surface` | 7.70:1 | 4.5:1 |
| oscuro | `--color-text-muted` | `--color-bg-subtle` | 6.71:1 | 4.5:1 |
| oscuro | `--color-text-on-action` | `--color-action-primary` | 6.75:1 | 4.5:1 |
| oscuro | `--color-text-on-action` | `--color-action-primary-hover` | 7.88:1 | 4.5:1 |
| oscuro | `--color-text-on-action` | `--color-action-primary-active` | 5.44:1 | 4.5:1 |
| oscuro | `--color-text-on-danger` | `--color-action-danger` | 7.69:1 | 4.5:1 |
| oscuro | `--color-text-on-danger` | `--color-action-danger-hover` | 9.30:1 | 4.5:1 |
| oscuro | `--color-feedback-danger` | `--color-bg-surface` | 7.21:1 | 4.5:1 |
| oscuro | `--color-feedback-danger` | `--color-feedback-danger-subtle` | 6.60:1 | 4.5:1 |
| oscuro | `--color-feedback-success` | `--color-bg-surface` | 7.88:1 | 4.5:1 |
| oscuro | `--color-feedback-success` | `--color-feedback-success-subtle` | 6.88:1 | 4.5:1 |
| oscuro | `--color-feedback-warning` | `--color-bg-surface` | 9.20:1 | 4.5:1 |
| oscuro | `--color-feedback-warning` | `--color-feedback-warning-subtle` | 8.01:1 | 4.5:1 |
| oscuro | `--color-feedback-info` | `--color-bg-surface` | 7.95:1 | 4.5:1 |
| oscuro | `--color-feedback-info` | `--color-feedback-info-subtle` | 7.24:1 | 4.5:1 |
| oscuro | `--color-focus-ring` | `--color-bg-canvas` | 10.33:1 | 3:1 |
| oscuro | `--color-focus-ring` | `--color-bg-surface` | 9.33:1 | 3:1 |
| oscuro | `--color-border-strong` | `--color-bg-canvas` | 3.86:1 | 3:1 |
| oscuro | `--color-border-strong` | `--color-bg-surface` | 3.48:1 | 3:1 |
| oscuro | `--color-border-accent` | `--color-bg-surface` | 6.33:1 | 3:1 |

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

**Responsive (obligatorio)**

Toda pantalla es **usable**, no solo visible, en cualquier dispositivo en que se trabaje: celular, tablet y PC de mostrador. Una pantalla que se rompe en algún ancho no está terminada.

- **Rango:** de 360px a 1920px de ancho, sin scroll horizontal de la página en ningún punto intermedio. Lo único que puede desplazarse a lo ancho es una tabla, dentro de su propio contenedor con `overflow-x: auto`.
- **Anchos de referencia:** 360 (celular), 768 (tablet vertical), 1024 (tablet horizontal), **1366×768** (el monitor típico del mostrador) y 1920. En 1366×768 la acción principal (por ejemplo, "Cobrar") se ve sin hacer scroll.
- **Orientación:** funciona en vertical y en horizontal. En un celular horizontal, con poca altura, la acción principal no queda tapada ni fuera de la pantalla.
- **Zoom y texto grande:** con zoom del navegador al 200% no se pierde contenido ni funciones (WCAG 1.4.4 y 1.4.10); el contenido se reacomoda en una columna. Por eso los tamaños van en `rem` y no se fija el alto de nada que contenga texto.
- **Táctil sin hover:** ninguna acción depende solo de `:hover`. Los estilos de hover van con `@include hover`, que solo aplica en dispositivos con hover; `@include hover(':disabled')` excluye un estado. El tamaño táctil lo resuelven los tokens `--size-control` y `--size-row`, que crecen solos con `pointer: coarse`.
- **Barras fijas en celular:** una barra fija abajo (total y "Cobrar") suma `env(safe-area-inset-bottom)` a su padding y no queda tapada por el teclado en pantalla. Usa `dvh` en vez de `vh` para el alto de la ventana.
- **Cómo se construye:**
  - Mobile-first: estilos base para el celular y `@include respond-to(md) { ... }` hacia arriba. Breakpoints `sm` 640, `md` 768, `lg` 1024, `xl` 1280.
  - Flex o grid con `gap`, con columnas que se apilan en pantallas angostas (`minmax(0, 1fr)`, `flex-wrap`). Un hijo con texto largo lleva `min-width: 0` para que no ensanche la página.
  - Propiedades lógicas (`margin-inline`, `padding-block`).
  - Imágenes y medios con `max-inline-size: 100%` (lo da el reset).
  - Nada con un ancho mínimo mayor que 360px.

**Accesibilidad (no negociable)**
- Contraste de texto 4.5:1 (3:1 para textos de 24px o más y para íconos). Verifícalo al definir los tokens, no después.
- `@include focus-ring` en todo lo interactivo. Nunca `outline: none` sin reemplazo.
- `@include touch-target` (44×44px mínimo) en botones y controles del POS.
- Texto solo para lectores de pantalla: clase global `.srOnly` o `@include visually-hidden` en el módulo.
- No transmitas información solo con color: un estado de error lleva ícono y texto.
- Las animaciones se envuelven en `@include reduced-motion-safe`.

**Temas**
- Claro por defecto; oscuro con `[data-theme='dark']` sobre `<html>` y fallback con `prefers-color-scheme` cuando no hay `data-theme='light'`. Los dos temas tienen exactamente los mismos tokens (lo comprueba el test).
- Como los componentes solo usan tokens semánticos, un tema nuevo no exige tocar ningún componente.

## 6. Componentes base (`shared/ui`)

Se importan desde `@/shared/ui`. El catálogo visual con todos los estados está en **`/dev/ui`** (solo en desarrollo: `npm run dev` y abre `http://localhost:5173/dev/ui`).

| Componente | Uso |
|---|---|
| `Button` | `variant` `primary` (la única acción naranja de la pantalla), `secondary`, `ghost` o `danger`; `size` `md` o `lg`; `loading` lo bloquea y anuncia `aria-busy`; deshabilitado se ve gris en todas las variantes; `icon` decorativo. Es `type="button"` por defecto. |
| `IconButton` | `label` obligatorio: es el nombre accesible y el tooltip. |
| `Field` | Label, control, ayuda (`hint`) y error. Asocia el label y describe el control con la ayuda y el error (`aria-describedby`, `aria-invalid`). El error lleva ícono y texto. Un control propio se conecta con `useFieldControl(props)`. |
| `Input` | `align="end"` para precios y cantidades. |
| `Select` | Radix Select: `options`, `value` u `defaultValue`, `onValueChange`, `placeholder`. |
| `Checkbox` | Nativo, con `label` y `hint` propios (no va dentro de `Field`). |
| `Table` | `Table` (con `caption` obligatorio), `TableHead`, `TableBody`, `TableRow` (`selected`), `TableHeaderCell` y `TableCell` (`numeric`: a la derecha y con cifras tabulares). Una línea por fila; en pantallas angostas la tabla se desplaza dentro de su contenedor. |
| `Dialog` | Radix Dialog: `title` y `description` obligatorios, `trigger`, `footer` con la acción principal al final y el cancelar envuelto en `DialogClose`. En el celular es una hoja inferior. |
| `useToast()` | `toast({ tone, title, description })`. `ToastProvider` ya está montado en `App.tsx`. Los errores no se cierran solos (quedan hasta que el usuario los cierra) e interrumpen al lector de pantalla; los demás desaparecen a los 5 u 8 segundos. |
| `Badge` | `tone` `neutral`, `success`, `warning`, `danger` o `info`. El texto debe decir el estado por sí solo. |
| `EmptyState` | `icon`, `title`, `description`, `action` y `headingLevel`. |
| `Spinner` / `Skeleton` | `Spinner` anuncia "Cargando…" (o su `label`); `decorative` si el control ya anuncia `aria-busy`. `Skeleton` es decorativo. |
| Íconos | `IconClose`, `IconCheck`, `IconChevronDown`, `IconPlus`, `IconSearch`, `IconAlert`, `IconInfo`, `IconSuccess` e `IconBox`: SVG propios de 1em en `currentColor`, siempre decorativos. Un ícono nuevo se agrega en `shared/ui/icons`. |

## 7. Particularidades de este producto

- **El POS es una herramienta de trabajo**, no una landing: prima la velocidad, la densidad legible y el teclado (lector de código de barras, atajos, Enter para confirmar). Las decoraciones son mínimas.
- **Las cifras mandan**: precios y cantidades alineados a la derecha, con números tabulares y separador de miles colombiano (`$ 12.500`).
- Usuarios de 35 a 60 años, a veces en mostradores con mala luz y pantallas modestas. El tamaño base de texto debe ser cómodo (16px como mínimo) y con alto contraste.
- Habrá tablets y celulares, así que los objetivos táctiles importan y toda pantalla cumple las reglas de **Responsive** de la sección 5.

## 8. Checklist antes de dar por terminado un componente

- [ ] Un `.module.scss` propio, con `.root` y sin estilos globales.
- [ ] Solo tokens semánticos y `npm run lint:styles` en verde.
- [ ] Estados hover, focus-visible, active, disabled y, si aplica, loading, error y vacío.
- [ ] Funciona con teclado y con lector de pantalla (rol, label y `aria-*`).
- [ ] Es usable en 360, 768, 1024, 1366×768 y 1920px, en vertical y horizontal donde aplique, y con zoom al 200%, sin scroll horizontal de la página.
- [ ] Se ve bien en tema claro y oscuro.
- [ ] Ninguna acción depende solo de hover.
- [ ] Tiene test de comportamiento (Testing Library) y, si es visual, revisión con captura de pantalla.
