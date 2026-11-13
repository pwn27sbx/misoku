# A20 · Editar carpetas en el editor

**Vas a aprender:** a crear, renombrar, mover y borrar archivos como si fueran líneas de texto, y a salir a mini.files cuando hace falta.

En el explorador, `m` abre la carpeta seleccionada como un archivo de líneas: una por entrada. Escribir una línea crea, editar un nombre renombra, borrar la línea borra, y `dd` con `p` en otra carpeta mueve. Nada toca el disco mientras escribes.

Al confirmar con `=`, Misoku enseña el plan: qué se crea, qué se renombra y qué se va a la Papelera, con los enlaces que habría que actualizar. Solo después de aceptar se aplica. Borrar manda a la Papelera del sistema, no destruye.

Desde el árbol, `i`, `a` y `o` abren el editor en la carpeta de la nota seleccionada con esa tecla ya recibida, para no perder el gesto. Y `M` entrega la pantalla a mini.files, el gestor de Neovim: al cerrarlo, Misoku relee la carpeta y aparece lo que hayas hecho.

## Tu turno

1. En el explorador crea la carpeta `Notas` con `botas.md` y `viejo.md` dentro, selecciona la carpeta y pulsa `m`.
2. En el editor de líneas crea `equipaje.md`, renombra `botas.md` a `calzado.md`, borra `viejo.md`, pulsa `=` y confirma.
3. En el árbol, pulsa `i`, `a` y `o` sobre una nota.
4. Pulsa `M`, crea `practica-mini.md` con mini.files, cierra y mira que aparece en Misoku.



Siguiente: [[A21 Desafío Avanzado]]
