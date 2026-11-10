# Después del tutorial

Esta nota reúne lo que no se practica dentro de las rutas: los comandos de terminal y las funciones que dependen de otros programas. Es de referencia: no tiene pasos.

## Comandos de terminal

- `misoku tutor` abre el tutorial y te deja en el primer paso que no hayas superado.
- `misoku tutor --route I` abre una ruta concreta (`B`, `I`, `A` o `E`).
- `misoku tutor --reset` mueve el tutorial a la papelera y empieza de cero.
- `misoku <carpeta>` abre otra carpeta, y `misoku --dir <carpeta>` hace lo mismo desde cualquier sitio.
- `misoku export nota.md -o salida.pdf` escribe una nota como PDF, también `:export` dentro de la app.
- `misoku paste-image` guarda en la nota una imagen del portapapeles desde fuera.
- `misoku sync setup` conecta el vault con GitHub y `misoku sync init` lo prepara a mano.
- `misoku --version` dice la versión, `misoku --help` lista las opciones, `misoku --list-themes` los temas y `misoku --theme <nombre>` elige uno.
- `misoku --themes-dir <carpeta>` lee temas propios, y la variable `MISOKU_WORKSPACE` fija la carpeta que se abre.

## Lo que se practica fuera

El sync con git, sus conflictos y los avisos de llegada necesitan dos equipos o un remoto real; el tutorial no inventa uno. Pasa lo mismo con el export a PDF, el diálogo de archivos del sistema, las variables de entorno y el marco de "terminal demasiado pequeño".

## Lo que solo se ve

Algunas cosas son solo visuales: el punto del tab con cambios sin guardar, el contador de palabras, `Ln N, Col M`, los colores de foco y el resaltado de sintaxis del preview. Se aprenden mirando, no con un paso.

## Seguir aprendiendo

Las rutas Intermedio, Avanzado y Experto cubren el resto: movimientos y selección, la línea de comandos, tablas, etiquetas, propiedades, diario, historial, grafo, plantillas, sync y todos los ajustes. Cada ruta asume la anterior, pero puedes entrar directo a la que te toque desde el inicio o con `misoku tutor --route`.
