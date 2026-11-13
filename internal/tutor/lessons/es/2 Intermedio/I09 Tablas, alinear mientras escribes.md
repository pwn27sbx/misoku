# I09 · Tablas: alinear mientras escribes

**Vas a aprender:** a mantener la tabla ordenada mientras escribes, sin contar espacios.

Una tabla se lee bien cuando todas las barras caen en la misma columna. `:tablefmt` (o `:tf`) rellena las celdas para conseguirlo de una vez. Y al escribir dentro de una celda con relleno, Misoku gasta ese relleno: la barra de cierre no se mueve mientras el texto cabe, y cuando ya no cabe, la columna se ensancha en todas las filas a la vez. El preview dibuja la tabla con bordes, sin los guiones del Markdown.

## Tu turno

1. Escribe una tabla de dos columnas con textos de distinto largo, pon el cursor dentro y ejecuta `:tablefmt` (o `:tf`).
2. Escribe `Playa` dentro de una celda con relleno: la barra de cierre no se mueve mientras cabe, y al pasarse la columna se ensancha.
3. Mira la tabla dibujada con bordes en el preview.

Siguiente: [[I10 Tablas desde el preview]]
