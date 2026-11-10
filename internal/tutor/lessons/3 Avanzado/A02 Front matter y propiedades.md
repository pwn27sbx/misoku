# A02 · Front matter y propiedades

**Vas a aprender:** a declarar los datos de una nota en su cabecera y a leerlos como una tabla en el preview.

El front matter es un bloque al principio del archivo, entre dos líneas de tres guiones, con pares de clave y valor. Misoku lo lee como un YAML sencillo: cada clave es una propiedad de la nota, no texto que se lea en el cuerpo.

Lo útil viene después: el preview convierte esas propiedades en una tabla bajo el encabezado, con una fila por clave. Así una nota de viaje enseña su destino y su estado sin abrir la cabecera.

`cover` e `icon` son la excepción. Son propiedades de presentación, se dibujan en la portada y en el explorador, y la tabla no las muestra para no repetirlas.

## Tu turno

1. Escribe al principio de esta nota, por encima del título, un bloque con `---`, `viaje: costa`, `estado: planeado` y otro `---`.
2. Mira la tabla de propiedades bajo el encabezado del preview.
3. Comprueba en esa tabla que no aparecen `cover` ni `icon`.



Siguiente: [[A03 Editar propiedades]]
