# E16 · Notas que llegan

**Vas a aprender:** enterarte de lo que trae el sync sin abrir nada.

La barra de estado cuenta el sync en cuatro estados cortos: `sincronizando…` mientras trabaja, `sincronizado hace 5 min` con la hora del último éxito, `N conflictos` si hay copias de conflicto sin resolver y `sin conexión · N pendientes` cuando no hay red y quedan commits esperando. Se leen de un vistazo y no interrumpen lo que estás escribiendo.

Cuando un sync trae notas nuevas, la barra nombra hasta tres y `:messages` guarda hasta diez, con su hora. Es el aviso de "esto ha llegado mientras no mirabas": si estabas en otra nota, ahí lo tienes.

Y solo una ventana sincroniza cada vault. Si abres una segunda instancia sobre las mismas notas, la nueva lo dice en la barra con `otra ventana sincroniza` y no compite por el repositorio: dos procesos empujando a la vez sería la forma más fácil de crear conflictos.

## Tu turno

1. Mira la barra: `sincronizando…`, `sincronizado hace 5 min`, `N conflictos`, `sin conexión · N pendientes`.
2. Haz que llegue una nota y lee el aviso: tres nombres en la barra, diez en `:messages`.
3. Abre una segunda ventana de Misoku sobre el mismo vault y mira qué dice la barra.



Siguiente: [[E17 Montar el sync]]
