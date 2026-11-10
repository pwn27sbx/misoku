# E19 · Vim de precisión

**Vas a aprender:** dos detalles que casi nadie usa y que se agradecen.

`ctrl+e` y `ctrl+y` desplazan la vista una línea hacia abajo o hacia arriba sin mover el cursor. Sirven para mirar lo que hay más abajo sin perder el punto donde estabas escribiendo: el cursor se queda en su línea y el texto se desliza debajo. Es el equivalente a asomarse por el borde de la página.

El historial de deshacer es por pestaña, no por aplicación. Cada nota tiene su propia pila, con hasta 200 cambios o 8 MB en memoria, y una pestaña nueva empieza vacía. Por eso, si editas en una nota, cambias a otra y pulsas `u`, no deshaces lo de la primera: en la segunda no hay nada que deshacer y la barra lo dice.

La consecuencia: puedes trabajar en dos notas a la vez sin que sus historiales se mezclen. Y si recargas una nota, su pila empieza de cero.

## Tu turno

1. Desplaza una línea con `ctrl+e` y `ctrl+y` sin mover el cursor.
2. Edita en una pestaña, cambia a otra y pulsa `u`: el historial de la segunda está vacío.



Siguiente: [[E20 Desafío Experto]]
