# A13 · Exportar a PDF

**Vas a aprender:** a convertir una nota en un PDF con el tamaño de una hoja, en la carpeta que quieras y con el nombre que quieras.

`:export` toma la nota de ahora, la dibuja como el preview pero paginada, y escribe el PDF junto al archivo. Si no le dices nada más, el nombre es el de la nota. Lo que no entra en una página sigue en la siguiente, y las tablas y las listas se parten donde toca.

Con un argumento cambias el destino. Una carpeta, terminada en barra, mete el PDF dentro con el nombre de la nota; un nombre de archivo lo escribe donde esté la nota con ese nombre. `:pdf` es el mismo comando.

`space P` hace lo mismo sin escribir nada, y funciona desde cualquier nota abierta. Es la vía rápida para mandar una nota por correo o imprimirla.

## Tu turno

1. Ejecuta `:export` en esta nota y mira el PDF que aparece a su lado.
2. Ejecuta `:export ~/Documentos/` (o una carpeta tuya) y luego `:export informe.pdf`.
3. Abre la nota `A00 Avanzado` y pulsa `space P`.



Siguiente: [[A14 Portada]]
