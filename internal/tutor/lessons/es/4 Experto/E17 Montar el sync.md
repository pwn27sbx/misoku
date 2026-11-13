# E17 · Montar el sync

**Vas a aprender:** de cero a un repositorio privado sincronizado.

`:sync setup`, o `s` en el inicio, es el asistente que monta todo: crea un repositorio privado en tu cuenta de GitHub, prepara la identidad de git, ejecuta `gh auth setup-git` para que las credenciales queden guardadas y añade el vault a `sync.vaults`. Al terminar, el vault tiene `.git`, tiene `origin` y aparece en la lista que sincroniza.

Si ya tienes el repositorio creado y prefieres la terminal, `misoku sync init <url>` hace el enlace sin asistente y resuelve tres casos: primer ordenador (repo vacío, sube tus notas), otro ordenador (repo con notas, las baja) y las dos cosas a la vez (notas en ambos lados: mezcla y conserva todo). El tercer caso es el que crea copias de conflicto en vez de elegir por ti.

Los dos caminos acaban en el mismo sitio: un vault de `sync.vaults` con su remoto. El asistente es para la primera vez; el comando, para repetirlo en otra máquina.

## Tu turno

1. Ejecuta `:sync setup` (o `s` en el inicio) y responde a sus preguntas.
2. Ejecuta `misoku sync init <url>` en una carpeta nueva y comprueba que el repositorio se crea y el vault queda listado.



Siguiente: [[E18 Grafo global a fondo]]
