# B22 · Buscar dentro de la nota

**Vas a aprender:** a buscar con `/`, a repetir con `n` y `N`, y a leer los mensajes de la búsqueda.

En NORMAL, `/` abre la búsqueda abajo; escribes, `Enter` salta a la primera coincidencia, `n` va a la siguiente y `N` a la anterior. Una búsqueda en minúsculas no distingue mayúsculas, así que `hotel` también encuentra `Hotel`; una mayúscula hace que la búsqueda sí las distinga, así que `Hotel` se salta `hotel`. Si no hay coincidencias la barra se pone roja con `E486: Patrón no encontrado`, y al llegar al final verás `la búsqueda llegó al FINAL, sigue desde el PRINCIPIO`, que avisa de que ha vuelto a empezar. La búsqueda funciona igual en el editor y en la vista de lectura.

## Tu turno

1. Pulsa `/`, escribe `hotel` y pulsa `Enter`; repite lo mismo en la vista de lectura.
2. Pulsa `n` para la siguiente coincidencia y `N` para la anterior.
3. Busca `Hotel` con mayúscula: encuentra `Hotel Miramar` y se salta cada `hotel` en minúsculas.
4. Busca `zzz` y lee el error; después busca `hotel` y pulsa `n` hasta pasar la última para ver el aviso de vuelta.

## Tu zona de pruebas

El hotel de la playa abre en junio y el desayuno se sirve hasta las diez.

El hotel del centro tiene bicicletas y una terraza con vistas al puerto.

Hotel Miramar, junto al casco antiguo, es el más barato de los tres.

Siguiente: [[B23 Pegar y equivocarse]]
