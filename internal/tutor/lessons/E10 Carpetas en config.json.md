# E10 · Carpetas en config.json

**Vas a aprender:** las tres carpetas que Misoku lee de la configuración.

`notes_dir` es la carpeta que Misoku abre cuando lo lanzas sin argumentos, como `misoku` a secas. Tiene que ser una ruta absoluta o empezar por `~`, porque una ruta relativa dependería de la carpeta desde la que arrancas. Cambiarla es lo que hace que tu icono del escritorio abra tu vault y no otro.

`daily_dir` y `templates_dir` son carpetas dentro del vault, no fuera. `daily_dir` decide dónde se escriben las notas del día (`AAAA-MM-DD.md`) y su valor de fábrica es `Diario`; `templates_dir` decide dónde busca `/Template` las plantillas, con `Plantillas` de fábrica. Misoku rechaza una ruta absoluta, una que salga del vault o la raíz del vault, y te dice por qué: así una errata no escribe notas fuera de tus notas.

Las dos se escriben igual: una ruta relativa, separada por `/`.

## Tu turno

1. Cambia `notes_dir` a otra carpeta y abre Misoku sin argumentos.
2. Pon `daily_dir` en una carpeta dentro del vault, por ejemplo `Diario/Viajes`; una carpeta de fuera no se acepta.
3. Pon `templates_dir` en `Plantillas` y comprueba que `/Template` lista sus notas.



Siguiente: [[E11 Sync en config.json]]
