# Guion de visita a negocios

Guía para las visitas de validación del MVP (`docs/negocio.md`, §8). La lista de negocios para visitar está en una hoja aparte (prospectos de ferreterías del nororiente de Medellín y de Bello).

**El objetivo de la visita es aprender, no vender.** Se quiere saber cómo trabaja hoy el negocio, qué le duele y si aceptaría ser piloto. La venta llega después, con el piloto.

---

## 1. Antes de salir

- [ ] Demo abierta y probada en el portátil o el celular, con la plantilla `ferreteria`. Como la demo todavía no está publicada en internet (llega en la fase 5), se lleva corriendo en local con datos ya cargados.
- [ ] Una venta de prueba hecha esa misma mañana, para no descubrir un error frente al dueño.
- [ ] La hoja de prospectos en el celular, con la ruta del día agrupada por zona (4 a 6 negocios por salida).
- [ ] El cronómetro del celular listo (pregunta P28).
- [ ] Copias impresas o en el celular de la **ficha de visita** (sección 7).
- **Horario:** de martes a jueves, entre 9:30 y 11:30 a. m. o entre 2 y 4 p. m. Evitar los sábados, los lunes temprano y la hora del almuerzo, que es cuando más venden los mostradores.

## 2. Mensaje de WhatsApp (si hay número publicado)

Corto, sin enlaces ni archivos adjuntos en el primer mensaje, para que no parezca spam:

> Buenos días, ¿hablo con la Ferretería [nombre]? Soy Brayan, vivo aquí cerca y estoy construyendo un sistema para ferreterías: ventas, inventario y factura electrónica. No le quiero vender nada todavía: busco ferreterías que me cuenten cómo trabajan para hacerlo a la medida. ¿Me regala 15 minutos esta semana? Puedo pasar por el local a la hora que menos lo afecte.

Si no responde en 3 días, se escribe una sola vez más:

> Hola de nuevo, soy Brayan, el del sistema para ferreterías. Si le queda más fácil, paso un momento por el local sin compromiso. ¿Qué día le sirve?

Sin respuesta después de eso: se marca "Visitar sin cita" y se pasa en persona.

## 3. Entrada en persona (30 segundos)

1. **Esperar a que no haya clientes en el mostrador.** Si hay fila, comprar algo pequeño y volver más tarde.
2. Preguntar por el dueño o el encargado.
3. Decir:

> Buenas, soy Brayan, vivo aquí en el sector. Estoy construyendo un programa para ferreterías y antes de terminarlo quiero entender cómo trabajan ustedes. No vengo a vender nada: son 15 minutos de preguntas y, si le interesa, le muestro cómo va. ¿Ahora es buen momento o paso otro día?

Si dice que no tiene tiempo: se pide un día y una hora concretos y se anotan en la hoja.

## 4. Preguntas (10 minutos)

Reglas para que las respuestas sirvan:
- Preguntar por **lo que pasó**, no por lo que harían ("¿cómo lo hizo la última vez?", no "¿usaría...?").
- No mostrar Cuadra hasta terminar las preguntas, para que las respuestas no se acomoden a la demo.
- Escuchar más de lo que se habla. Anotar las frases textuales que llamen la atención: van a la columna "Cita textual" de `docs/negocio.md`, §8.

**El negocio**
1. ¿Cuánto tiempo lleva el negocio? ¿Cuántas personas atienden el mostrador?
2. ¿Tiene otra sede o bodega?
3. Más o menos, ¿cuántas ventas hace en un día normal? ¿Y un sábado?

**Cómo vende hoy**
4. ¿Cómo registra una venta? (Cuaderno, talonario, Excel, un programa... ¿Cuál?)
5. Muéstreme cómo lo hace con la próxima venta. → **Cronometrar una venta real** (P28). Contar cuántos productos tenía.
6. ¿Cómo sabe el precio de cada producto? ¿Quién lo cambia cuando sube?
7. Cuando el total da con centavos o con pesos sueltos, ¿cómo redondea? ¿A 50, a 100? ¿Quién asume la diferencia? (P27)
8. ¿Vende por metro, por kilo o fraccionado? ¿Cómo lo cobra?

**Inventario**
9. ¿Cómo sabe qué se le está acabando? ¿Cuándo fue la última vez que se le acabó algo que pedían?
10. ¿Cuándo fue la última vez que contó el inventario? ¿Cuánto se demoró? ¿Le cuadró?
11. ¿Cuántos productos distintos maneja, más o menos? ¿Los tiene en alguna lista o Excel?

**Factura electrónica (DIAN)**
12. ¿Ya factura electrónicamente? ¿Con qué programa o proveedor? ¿Cuánto paga al mes?
13. ¿Cuántas facturas electrónicas emite al mes, más o menos? (**pregunta crítica** de `docs/negocio.md`, §2)
14. Si no factura todavía: ¿la DIAN ya le exigió? ¿Qué piensa hacer?

**Caja y fiados**
15. ¿Cómo cierra la caja al final del día? ¿Le pasa que no cuadra? ¿Qué hace entonces?
16. ¿Fía? ¿Cómo lleva la cuenta de quién le debe?
17. ¿Tiene clientes que compran mucho (maestros de obra, contratistas) a un precio distinto?

**Lo que más le duele**
18. De todo esto, ¿qué es lo que más tiempo o plata le hace perder?
19. Si usa un programa: ¿qué es lo que más le molesta de él? ¿Lo ha pensado cambiar? ¿Por qué no lo ha hecho?

## 5. Demo (5 minutos, solo si hay interés)

Mostrar **solo lo que responde a lo que dijo** en las preguntas. No hacer un recorrido completo.

| Si dijo que le duele... | Mostrar |
|---|---|
| Lo lento que es vender | Una venta de 5 productos en el POS, con el escáner o la búsqueda. Que la cronometre él mismo. |
| Vender fraccionado | Vender 2,5 metros de un producto que se vende por metro. |
| No saber qué falta | El stock mínimo y la lista de productos por agotarse. |
| Contar el inventario | La toma física. |
| Los fiados | Una venta a crédito y el estado de cuenta del cliente. |
| Que la caja no cuadra | La apertura, el cierre y el arqueo por billetes y monedas. |
| La DIAN | Una venta que genera su documento electrónico. **Aclarar que todavía está en modo de prueba.** |
| Pasar sus productos | La importación desde Excel. |

Después, preguntar: **"¿Qué le falta a esto para que usted lo usara todos los días?"**

## 6. Cierre

- **Si mostró interés:** ofrecer el piloto.
  > Voy a arrancar con 3 a 5 ferreterías. Durante el piloto no pagan nada: yo les paso los productos, los capacito y los acompaño la primera semana. A cambio, me cuentan qué funciona y qué no. Después, quienes estuvieron en el piloto tienen precio especial de fundador. ¿Le gustaría estar en el grupo?
- **Pedir algo concreto**, que es la prueba real de interés: el Excel o la lista de productos, una segunda cita o el contacto del encargado. Un "sí, me interesa" sin nada más **no cuenta** como piloto interesado.
- **Pedir un referido:** "¿Conoce otra ferretería a la que le pueda servir?"
- Agradecer y cumplir lo que se prometa (si se dijo "le escribo el jueves", escribir el jueves).

**Lo que no se hace:**
- Prometer fechas de arranque. El piloto empieza al terminar la fase 5.
- Prometer funciones que no existen. Lo que pidan se anota como hallazgo.
- Decir precios. Todavía no están definidos (`docs/negocio.md`, §1). Si insisten: "estará por debajo de lo que pagan hoy entre el programa y la factura electrónica, y el piloto es gratis".
- Hablar mal del programa que usan hoy.

## 7. Ficha de visita

Una por negocio. Se llena **en los 10 minutos siguientes a la visita**, antes de entrar al próximo negocio. El resumen se copia en la hoja de prospectos y los hallazgos en `docs/negocio.md`, §8.

```
Negocio:                      Fecha:            Zona:
Habló con (rol, no nombre):   dueño / encargado / cajero
Personas en el mostrador:     Sedes:            Ventas/día (aprox.):
Cómo registra hoy:            cuaderno / talonario / Excel / programa: ________
Factura electrónica:          sí (proveedor: ______, $____/mes) / no
Documentos DIAN al mes:
Tiempo de una venta (P28):    ___ s con ___ productos
Redondeo de centavos (P27):
Vende fraccionado:            sí / no         Fía: sí / no
Productos distintos (aprox.):  ¿Tiene lista o Excel?: sí / no
Lo que más le duele:
Frases textuales:
Qué le gustó de la demo:
Qué le faltó:
¿Piloto?:                     sí, con compromiso (______) / tal vez / no
Próximo paso y fecha:
Referidos:
```

## 8. Después de cada salida

1. Actualizar el Estado de cada negocio en la hoja.
2. Sumar los hallazgos repetidos en la tabla de `docs/negocio.md`, §8 (columna "Veces").
3. Después de unas 10 visitas: revisar si las preguntas siguen sirviendo y ajustar este guion.
