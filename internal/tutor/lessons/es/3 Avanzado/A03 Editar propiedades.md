# A03 · Editar propiedades

**Vas a aprender:** a cambiar un valor desde la tabla del preview y a saber qué propiedades se editan ahí y cuáles solo en el editor.

Un valor suelto se edita en la tabla con un clic: se abre un campo con el valor de ahora, lo cambias y el clic se convierte en un solo cambio deshacible, como si lo hubieras escrito tú. La clave no se toca, así que `viaje` sigue escrito `viaje` aunque el valor cambie.

Las listas son distintas. Una propiedad como `tags: [playa, sol]` se dibuja de solo lectura: la tabla no sabe si quieres añadir, quitar o reordenar, y por eso ese cambio se hace en el editor, escribiendo la lista a mano.

Esta diferencia importa: la tabla es para retocar un dato, el editor es para reescribir la estructura.

## Tu turno

1. Al principio de esta nota escribe un bloque con `---`, `viaje: costa` y otro `---`; luego haz clic en el valor `costa` de la tabla, cámbialo por `sierra` y confirma.
2. Añade al front matter `tags: [playa, sol]`, intenta editar la lista desde la tabla y, al ver que no se puede, añade `montaña` en el editor.



Siguiente: [[A04 Estado de la nota]]
