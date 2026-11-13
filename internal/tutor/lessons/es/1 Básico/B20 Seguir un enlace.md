# B20 · Seguir un enlace

**Vas a aprender:** las cuatro formas de saltar a otra nota, cuáles abren una pestaña nueva, cómo volver con `ctrl+o` y qué pasa si la nota no existe.

Con el cursor en un enlace y en NORMAL, `gd` lo abre en la misma pestaña, como en Neovim, y `gx` hace lo mismo; si tu pestaña tiene cambios sin guardar, se conserva y la nota se abre al lado. `Enter`, como en LazyVim, y un clic desde el preview la abren en una pestaña nueva. Si la nota ya está abierta, solo pasas a su pestaña. Si el destino no existe, `gd` y `gx` abren un borrador con ese nombre para que lo escribas, mientras que `Enter` solo avisa de que no hay nota. Esa diferencia importa: `Enter` nunca crea una nota.

`ctrl+o` devuelve la pestaña a la nota anterior, en la línea donde estaba el cursor, y cada pulsación retrocede una más. No funciona en INSERT.

## Tu turno

1. Con el cursor en el enlace de la zona de pruebas, pulsa `gd`.
2. Vuelve con `ctrl+o` y pulsa `gx` sobre el mismo enlace.
3. Pulsa `Enter` sobre el enlace.
4. Haz clic en el enlace desde el preview.
5. Escribe `[[Lista de playa]]` en una línea nueva y pulsa `gd`: la nota no existe y se abre un borrador.

## Tu zona de pruebas

Lee: [[Después del tutorial]] reúne los comandos de terminal y lo que no se practica en las rutas.

Para volver a esta nota después de saltar: si fue en la misma pestaña, pulsa `ctrl+o`; si se abrió una pestaña nueva, usa `H` o la pestaña de la lección en la barra.

Siguiente: [[B21 Leer sin editar]]
