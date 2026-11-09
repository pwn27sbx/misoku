# A16 · Formularios: opciones y estrellas

**Vas a aprender:** las tres formas de responder sin escribir: una opción, varias opciones y una valoración de estrellas.

Una pregunta de elección única se escribe con el tipo `(choice)` y sus opciones separadas por barras: `? ¿Dónde duermes? (choice): Cabaña | Hotel | Apartamento`. En el preview se contesta con un clic; otro clic en la misma opción borra la respuesta, y por eso se puede cambiar de idea sin borrar nada a mano.

La elección múltiple es igual con el tipo `(multi)`. Cada clic suma o quita una opción, y la respuesta las guarda unidas por comas en el orden en que están escritas, no en el que las pulsaste.

La valoración es el tipo `(rating)`: cinco estrellas y una respuesta con el número. La tercera estrella escribe `3`. Las tres formas escriben la respuesta en la línea de debajo, así que el resultado se lee en el texto y se puede corregir a mano.

## Tu turno

1. Inserta un formulario con `/Form` y escribe una pregunta de elección con tres opciones (`Cabaña`, `Hotel`, `Apartamento`); marca `Cabaña`, bórrala con otro clic y vuelve a marcarla.
2. Añade una pregunta múltiple con `Bici`, `Tren` y `Coche`, y marca dos opciones.
3. Añade una pregunta de valoración y pulsa la tercera estrella.



Siguiente: [[A17 Formularios, editarlos]]
