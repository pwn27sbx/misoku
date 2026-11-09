# A09 · Incrustar

**Vas a aprender:** a traer el contenido de otra nota, de una sección, de un bloque o de una imagen, sin copiarlo.

Un enlace normal lleva a otra nota. Si le pones una exclamación delante, el contenido viene aquí: `![[Viaje]]` dibuja la nota entera dentro de esta, y si `Viaje` cambia, esto cambia con ella.

El molde es el de siempre: `![[Nota#parte]]`. El `#` separa la nota de la parte, y `^` dice que la parte es un bloque y no un título. Por eso `![[Viaje#Equipaje]]` trae la sección bajo ese título y `![[Viaje#^reserva]]` trae solo la línea marcada con `^reserva`. En la misma nota se omite el nombre: `![[#^reserva]]`. Sin el `#`, `[[^reserva]]` sería una nota llamada `^reserva`, y por eso el `#` no es opcional: es la frontera entre la nota y la parte.

Con una imagen pasa lo mismo: `![[mapa.png]]` la dibuja por su nombre, sin ruta. Es la forma de tener una foto o un plano dentro de la nota sin depender de dónde esté el archivo.

## Tu turno

1. Crea la nota `Viaje` con un título `Equipaje` y una línea que termine en `^reserva`; después escribe aquí `![[Viaje]]` en una línea nueva.
2. Escribe `![[Viaje#Equipaje]]` y mira que trae solo esa sección.
3. Escribe `![[Viaje#^reserva]]` y mira que trae solo ese bloque.
4. Escribe `![[mapa.png]]` con una imagen que ya esté en el vault, o con el nombre de la que tengas.



Siguiente: [[A10 Enlaces a archivos y enlaces peligrosos]]
