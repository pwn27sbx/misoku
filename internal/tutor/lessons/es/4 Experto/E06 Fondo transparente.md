# E06 · Fondo transparente

**Vas a aprender:** a dejar ver el terminal detrás de la app y a dibujar una casilla en curso.

`ui.transparent` deja sin pintar los fondos base: editor, preview, explorador, línea de pestañas y barra de estado. Lo que sí se pinta son los resaltados y las cajas flotantes. El resultado se ve con un terminal que tenga opacidad o desenfoque: el fondo del terminal aparece detrás de Misoku.

La fila "Fondo transparente" de `:settings` lo activa y lo apaga. Al apagarlo no escribe `false`: borra la clave, porque apagado es el valor de fábrica. También puedes escribir `"transparent": true` a mano dentro de `ui`, y el fondo cambia al reiniciar.

Y ya que estás en `ui`, hay un ajuste que solo se puede poner a mano: `ui.checkbox.todo`. Es el glifo que dibuja una casilla en curso, la que se escribe como `- [-]`. Sin glifo, `[-]` es texto normal; con él, el preview pinta la casilla en amarillo con tu símbolo. Hay que reiniciar Misoku para verlo.

## Tu turno

1. Activa la fila Fondo transparente y apágala después.
2. Comprueba lo mismo escribiendo `ui.transparent` a mano dentro de `ui`.
3. Añade `ui.checkbox.todo` con el glifo que quieras y reinicia Misoku.
4. Escribe en esta nota una tarea en curso, por ejemplo `- [-] llamar al hotel`, y mírala en el preview.



Siguiente: [[E07 Números y cursor]]
