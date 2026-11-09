# E14 · El estado del sync

**Vas a aprender:** la pantalla que cuenta todo lo que el sync está haciendo.

`:sync status` abre una ventana de lectura con el estado completo: si está sincronizando o en reposo, el remoto (con la credencial tapada), la rama, cada cuánto sincroniza, cuántos commits quedan por empujar, cuándo fue el último sync bueno, el último error si lo hubo y las copias de conflicto que hay que resolver.

Abajo aparecen dos listas que explican lo raro. "Files that do not sync as they are" son los archivos que git no puede subir tal cual: más de 50 MB, ignorados por `.gitignore`, con nombres que Windows no acepta o dentro de otro repositorio anidado. Y "Arrived" recuerda las notas que trajo el último sync, por si llegaron mientras mirabas otra cosa.

Es la primera parada cuando algo no cuadra: casi siempre el motivo está escrito ahí antes de que lo busques en otra parte.

## Tu turno

1. Ejecuta `:sync status` y lee estado, remoto, rama, intervalo, pendientes, último error y conflictos.
2. En esa misma pantalla, mira la lista de archivos que quedan fuera: grandes, ignorados o con nombres ilegales.



Siguiente: [[E15 Cuando se detiene]]
