# A05 · Filtros avanzados

**Vas a aprender:** a combinar el estado, la carpeta y una palabra en una sola búsqueda, y a llegar a ese filtro desde el comando `:status`.

El buscador entiende condiciones además de palabras. `status:active` deja solo las notas activas, y `folder:Viajes` deja solo las que viven en esa carpeta, en cualquier punto de su ruta. Una carpeta con espacios se escribe entre comillas: `folder:"Mis notas"`.

El comando `:status` es un atajo: abre el buscador ya filtrado por un estado, o te ofrece los cuatro para elegir. Es la forma rápida de ver el tablero del día.

Lo bueno llega al combinar: todas las condiciones deben cumplirse y, además, la palabra tiene que aparecer en el nombre o en el texto de la nota. `status:active folder:Viajes playa` responde a una pregunta muy concreta sin abrir una sola nota.

## Tu turno

1. Ejecuta `:status` y elige uno de los cuatro estados.
2. Abre el buscador y escribe `status:active`.
3. Escribe `folder:Viajes` y luego `folder:"Mis notas"`.
4. Combina las tres cosas en una búsqueda: `status:active folder:Viajes playa`.



Siguiente: [[A06 Historia de la nota]]
