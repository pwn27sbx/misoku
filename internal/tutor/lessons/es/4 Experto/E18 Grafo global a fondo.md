# E18 · Grafo global a fondo

**Vas a aprender:** las teclas que faltaban del grafo global.

El grafo global (`space G`, `:graph all` o `g` en el inicio) dibuja todo el vault y sus enlaces. Se recorre sin ratón: `h`, `j`, `k` y `l` mueven la selección entre nodos, `n` y `N` saltan al siguiente y al anterior, `Enter` abre el nodo, `+` y `-` acercan y alejan, `0` vuelve a centrar y `Esc` limpia el filtro o vuelve a la selección anterior.

`o` oculta los nodos huérfanos y `u` los que faltan, esos círculos huecos de notas que aún no existen. Son los dos interruptores que convierten una maraña en un mapa legible.

`/` filtra el grafo por nombre de nota: no busca dentro de una nota, cambia qué nodos se dibujan. Y como es un argumento del comando, `:graph foo` no es un filtro: es un error de sintaxis y la barra responde `E488: Sobran caracteres`. El filtro se activa con `/` dentro del grafo.

## Tu turno

1. En el grafo global usa `h j k l`, `n`/`N`, `Enter`, `+`/`-`, `0`, `Esc`, `o` y `u`.
2. Pulsa `/` y comprueba que filtra el grafo en vez de buscar dentro de una nota.
3. Ejecuta `:graph foo` y lee `E488: Sobran caracteres` en la barra.



Siguiente: [[E19 Vim de precisión]]
