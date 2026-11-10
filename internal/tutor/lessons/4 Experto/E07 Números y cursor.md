# E07 · Números y cursor

**Vas a aprender:** numerar los títulos y cambiar la forma del cursor.

"Heading numbers" numera los títulos del preview como un índice: el primer título de nivel uno es `1`, sus hijos `1.1` y `1.2`, el siguiente `2`. Sustituye a los iconos de nivel, así que si tenías Numbers en Icons & bullets, verás la numeración en su lugar. Sin la clave, la numeración está encendida; apagarla escribe `"heading_numbers": false` dentro de `ui` y encenderla la deja en `true`.

`ui.cursor_shape: auto` hace que el cursor dibujado siga al modo, como en Neovim: bloque en NORMAL y en VISUAL, barra fina en INSERT. Con `block`, el cursor es siempre un bloque. La variable de entorno `MISOKU_CURSOR_SHAPE`, si existe, manda sobre el archivo; es una de esas cosas que se ajustan antes de abrir la app.

Con `auto` tienes una pista constante del modo en el que estás, sin mirar la barra de estado.

## Tu turno

1. Enciende y apaga "Heading numbers" y déjalo apagado para que el cambio quede guardado.
2. Con `ui.cursor_shape: auto`, mira el bloque en NORMAL y la barra al entrar en INSERT.



Siguiente: [[E08 El selector de temas]]
