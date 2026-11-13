# E15 · Cuando se detiene

**Vas a aprender:** los tres frenos del sync y cómo se resuelven.

Cuando el sync no puede seguir, Misoku para y lo explica en palabras llanas, sin soltar un error de git en crudo. El diálogo dice qué pasó y ofrece "Comprobar de nuevo" para reintentar cuando lo hayas arreglado; no se queda reintentando solo ni pierde lo que ya había hecho.

El freno más común es un candado de git huérfano: un `index.lock` que quedó de un proceso interrumpido. Mientras exista, git se niega a trabajar. El diálogo ofrece "Quitar el bloqueo y sincronizar", que borra ese archivo y sigue donde estaba.

El otro es la credencial de GitHub caducada en un remoto por HTTPS: en vez de fallar en silencio, la app ofrece "Iniciar sesión", que ejecuta `gh auth login`. Aceptas, te autenticas en la terminal y el siguiente sync funciona.

Ninguno de los tres se resuelve solo: la app pregunta antes de tocar nada.

## Tu turno

1. Pulsa `space s` con un problema simulado, lee el diálogo y usa "Comprobar de nuevo".
2. Deja un candado de git y acepta "Quitar el bloqueo y sincronizar": el archivo desaparece y el sync sigue.
3. Con un remoto de GitHub caducado, elige "Iniciar sesión" para ejecutar `gh auth login`.

Apunta aquí abajo qué freno te salió y cómo lo resolviste.



Siguiente: [[E16 Notas que llegan]]
