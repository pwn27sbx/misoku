# A04 · Estado de la nota

**Vas a aprender:** a darle un estado a cada nota y a reconocerlo de un vistazo, sin abrirla.

El estado es una propiedad más: `status:` con uno de cuatro valores. `active` es una nota en marcha, `on hold` está aparcada, `completed` está terminada y `dropped` se abandonó. Se escriben en el front matter y se guardan como cualquier dato.

En cuanto el estado está guardado, el explorador y el inicio cambian el icono de la nota por un glifo: un punto para lo activo, medio círculo para lo aparcado, una marca para lo terminado y una cruz para lo abandonado. Así el árbol se lee como un tablero.

Un valor que no sea uno de esos cuatro no es un error: la nota se trata como si no tuviera estado y vuelve su icono normal. Sirve para experimentar sin romper nada.

## Tu turno

1. Añade `status: active` al front matter y guarda la nota.
2. Cambia el estado a `on hold`, a `completed` y a `dropped`, guardando cada vez, y mira el glifo en el explorador y en el inicio.
3. Escribe `status: cualquier cosa`, guarda y comprueba que la nota se queda sin glifo.



Siguiente: [[A05 Filtros avanzados]]
