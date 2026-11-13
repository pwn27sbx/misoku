# I27 · Hoy y plantillas

**Vas a aprender:** a escribir la nota del día y a reutilizar plantillas con datos.

La nota del día se llama como la fecha y vive en `Diario/`: `space o d` (o `:daily`) abre la de hoy, y el primer guardado la crea. Una plantilla es una nota normal en `Plantillas/` que insertas con `/Template`; lo que trae cae en el cursor. Dentro de la plantilla, `{{fecha}}` se cambia por la fecha de hoy y `{{titulo}}` por el nombre de la nota donde la insertas. Las propiedades son lo más útil: las claves que la nota ya tiene se respetan y solo se añaden las que faltan, así que una plantilla puede traer `lugar:` sin pisar tu `estado:`.

Para esta lección, añade al principio de la nota un bloque de propiedades con `estado: en curso`.

## Tu turno

1. Pulsa `space o d` (o `:daily`) y guarda la nota del día.
2. Crea la nota `Plantillas/diario` con dos líneas de texto: `Plan del día` y `Sin prisa`.
3. En esta nota, inserta `/Template` y elige `diario`.
4. Añade a la plantilla dos líneas, `Fecha: {{fecha}}` y `Nota: {{titulo}}`, y vuelve a insertarla aquí.
5. Añade `estado: nuevo` y `lugar: playa` a las propiedades de la plantilla e insértala en esta nota.

Siguiente: [[I28 El explorador que se pliega solo]]
