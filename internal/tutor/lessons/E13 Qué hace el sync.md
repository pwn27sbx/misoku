# E13 · Qué hace el sync

**Vas a aprender:** los dos momentos en los que git trabaja sin que lo pidas.

Guardar una nota no escribe el commit en el acto. Unos segundos después de tu último guardado, Misoku hace un commit local con los cambios: nada de red, solo `git commit`. Así el historial avanza mientras escribes, y un sync posterior solo tiene que empujar lo que ya está commiteado.

Al cerrar Misoku pasa lo contrario: la app empuja lo pendiente durante un máximo de diez segundos después de que la pantalla desaparezca. Si el remoto responde, tus notas salen solas; si no, quedan en commits locales esperando al siguiente sync, y la barra lo contará al volver a abrir.

La consecuencia práctica: cerrar la app nunca pierde trabajo, pero no siempre lo publica. Si quieres estar seguro, abre, mira el estado y deja que termine.

## Tu turno

1. Guarda una nota y espera unos segundos: en el historial de git hay un commit local con ese cambio.
2. Cierra Misoku y mira que empuja hasta diez segundos después de cerrar la pantalla.

Anota aquí abajo lo que veas en la barra; así el tutorial sabe que has pasado por el paso.



Siguiente: [[E14 El estado del sync]]
