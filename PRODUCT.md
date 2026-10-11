# PRODUCT.md — contexto de producto para el diseño de UI

> Borrador escrito a mano en T13 (no generado con `/impeccable init`). Fuentes: `docs/negocio.md`, `docs/estilos.md` ("Particularidades de este producto") y `CLAUDE.md`. Las afirmaciones marcadas con **[validar]** salen de suposiciones y se confirman en las visitas a ferreterías.

## Qué es

Cuadra es el sistema del mostrador de una ferretería colombiana pequeña o mediana: vender (POS), saber qué hay en la bodega (inventario y kardex), fiar a los clientes de confianza y facturar electrónicamente ante la DIAN. El nombre viene de "cuadrar la caja": al cierre del día, las cuentas tienen que dar.

## Quién lo usa

| Rol | Quién es | Dónde y cómo lo usa | Lo que más le importa |
|---|---|---|---|
| **Cajero** (`cashier`) | Empleado de mostrador, a menudo el hijo o sobrino del dueño, o un trabajador de años | De pie, con fila de maestros de obra esperando. PC de mostrador o tablet, lector de código de barras, a veces una sola mano libre | Vender rápido sin equivocarse en el precio ni en el cambio |
| **Dueño** (`owner`) | 35 a 60 años, conoce el negocio de memoria, no necesariamente cómodo con software | En el mostrador en horas pico y en la oficina o el celular al cierre | Que la caja cuadre, cuánto vendió, qué se está acabando, quién le debe |
| **Administrador** (`admin`) | Persona de confianza que maneja precios, compras y usuarios | Escritorio, en ratos de menos movimiento | Cargar productos y precios sin errores, corregir inventario |
| **Bodeguero** (`warehouse`) | Recibe mercancía y despacha | En la bodega, con celular o tablet, manos sucias o con guantes **[validar]** | Registrar entradas rápido y encontrar un producto por código |

## Contexto físico (condiciona el diseño)

- **Luz mala y pantallas modestas:** monitores viejos de 1366×768, tablets económicas, celulares de gama media. Hay reflejos y polvo. → Alto contraste, texto base de 16px como mínimo, nada que dependa de matices sutiles de color.
- **Ruido e interrupciones:** el cajero atiende, contesta el teléfono y vuelve. → El estado de la venta en curso tiene que ser obvio de un vistazo; nada se pierde si se aleja.
- **Teclado y lector primero en el POS:** el lector de código de barras "teclea" y envía Enter. Atajos de teclado para lo frecuente. El mouse es secundario.
- **Táctil en tablet y celular:** objetivos de 44px como mínimo y separación suficiente para dedos gruesos.
- **Conexión inestable** **[validar]**: la UI debe dejar claro cuándo algo no se guardó.

## Qué significa "bien hecho" para ellos

- Una venta de 5 productos en 20 segundos o menos (meta del piloto).
- Las cifras se leen sin esfuerzo: precios y cantidades alineados a la derecha, números tabulares, separador de miles colombiano (`$ 12.500`), total grande.
- Cero sorpresas: un error dice qué pasó y qué hacer, en español llano ("No hay stock suficiente de Cemento gris 50kg: quedan 3").
- Se aprende en una tarde de capacitación.

## Tono de la marca

- **Confiable y sobrio**, como una buena herramienta: no juguete, no banco. Habla como un colega del gremio, de "usted" o neutro, sin tecnicismos ni anglicismos ("Cobrar", no "Checkout").
- **Directo:** verbos de acción en los botones ("Cobrar", "Registrar entrada", "Guardar precio").
- **Sin decoración gratuita:** ilustraciones, degradados y animaciones solo si ayudan a entender algo. La personalidad sale de la tipografía, la jerarquía y un color de marca bien usado.
- **Colombiano sin folclor:** formatos locales (COP, fechas `dd/mm/aaaa`, hora de Bogotá) sin caer en clichés visuales.

## Principios de diseño

1. **Las cifras mandan.** El total, el cambio y el stock son los datos más importantes de cada pantalla y se diseñan primero.
2. **Densidad legible.** Mucha información por pantalla (es una herramienta de trabajo) sin sacrificar el tamaño mínimo del texto.
3. **El color significa algo.** El color de marca marca la acción principal; rojo, ámbar y verde son estados, y siempre van con ícono o texto.
4. **Rápido antes que bonito, pero las dos cosas.** La calidad visual es parte del producto y del portafolio, nunca a costa de la velocidad.
5. **Claro y oscuro de primera clase.** El tema oscuro sirve en mostradores con mucho reflejo o en la noche, y cumple el mismo contraste.

## Fuera de alcance del diseño (por ahora)

- Marketing, landing page y branding completo (logo definitivo).
- Personalización visual por tenant (colores propios de cada ferretería).
