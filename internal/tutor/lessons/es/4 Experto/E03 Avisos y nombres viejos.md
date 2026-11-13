# E03 · Avisos y nombres viejos

**Vas a aprender:** qué avisa Misoku cuando un atajo choca y por qué los nombres viejos siguen funcionando.

Un atajo mal puesto no rompe nada: Misoku lo revisa al arrancar y te avisa. Hay tres choques típicos: la misma tecla en dos acciones, una tecla del explorador que nunca llega porque otra acción la captura antes, y un `space o` que es el principio de una secuencia más larga. El aviso sale al arrancar y queda en `:messages` para que puedas releerlo sin prisa.

Los nombres de las acciones cambiaron en la versión 0.27: `reading_mode` es ahora `reading_view` y `toggle_preview` es `editor_view`. Los nombres viejos se siguen aceptando y hacen lo mismo, así que una configuración antigua no se queda sin atajos al actualizar.

Lo correcto es usar los nombres nuevos; los viejos están solo para no romper lo que ya tenías.

## Tu turno

1. Da la misma tecla a dos acciones, reinicia y lee el aviso al arrancar y en `:messages`.
2. En `keys`, usa los nombres viejos `reading_mode` y `toggle_preview` y comprueba que activan la vista de lectura y el editor.



Siguiente: [[E04 El menú settings]]
