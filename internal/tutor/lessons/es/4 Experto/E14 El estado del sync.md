# E14 · El estado del sync

**Vas a aprender:** la pantalla que cuenta todo lo que el sync está haciendo.

`:sync status` abre una ventana de lectura con el estado completo, un campo por línea: Estado (las mismas palabras que la barra de estado), Remoto (con la credencial tapada), Rama, Sincronización periódica (cada cuánto sincroniza), Pendientes (commits que quedan por enviar), Última sincronización (cuando esta ventana ya sincronizó), Último error y las Copias en conflicto que hay que resolver.

Abajo aparecen dos listas que explican lo raro. "Archivos que no se sincronizan tal cual" son los archivos que git no puede subir tal cual: más de 50 MB, ignorados por `.gitignore`, con nombres que Windows no acepta o dentro de otro repositorio anidado. Y la lista de notas que trajo el último sync, como "2 notas nuevas desde otro equipo", te recuerda lo que llegó, por si llegó mientras mirabas otra cosa.

Es la primera parada cuando algo no cuadra: casi siempre el motivo está escrito ahí antes de que lo busques en otra parte.

## Tu turno

1. Ejecuta `:sync status` y lee Estado, Remoto, Rama, Sincronización periódica, Pendientes, Última sincronización, Último error y Copias en conflicto.
2. En esa misma pantalla, mira la lista de archivos que quedan fuera: grandes, ignorados o con nombres ilegales.



Siguiente: [[E15 Cuando se detiene]]
