# E02 · Cambiar tus atajos

**Vas a aprender:** a reasignar una tecla y a desasignar otra.

Todos los atajos viven en `keys`, dentro de `config.json`. Cada acción guarda la lista de teclas que la disparan, así que puedes añadir una tecla a una acción, quitarle otra o dejarla sin ninguna. El nombre de la acción es el que ves en la paleta de comandos.

Una lista vacía es la forma de decir "esta acción no tiene atajo": `[]`. La acción sigue existiendo en la paleta y se puede ejecutar desde ahí, pero ninguna tecla la dispara. Es útil cuando una tecla te molesta y prefieres pulsar la acción a mano.

Los cambios se leen al arrancar. Guarda el archivo, cierra Misoku y vuelve a abrirlo para probar la tecla nueva.

## Tu turno

1. En `config.json`, dentro de `keys`, dale otra tecla a `reading_view`, por ejemplo `z`: `"reading_view": ["space p", "z"]`. Reinicia y comprueba que `z` abre la vista de lectura.
2. A otra acción, por ejemplo `export_pdf`, ponle `[]` y comprueba que deja de responder a su tecla aunque siga en la paleta.

Si el JSON tiene una coma de más, Misoku arranca con los atajos de fábrica y avisa en `:messages`; la lección siguiente trata justo de esos avisos.



Siguiente: [[E03 Avisos y nombres viejos]]
