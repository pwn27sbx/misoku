# E09 · Tema por nombre y temas propios

**Vas a aprender:** a fijar un tema sin abrir la lista y a añadir paletas tuyas.

`:colo <nombre>` aplica ese tema y lo guarda en `config.json` de una vez; no hace falta abrir el selector. Es la forma de dejar Misoku con un tema concreto desde el primer arranque, y la que puedes repetir en otra máquina. Los nombres son los de la lista: `onedark`, `catppuccin`, `zenburn`.

Los temas propios son archivos JSON con una paleta, y viven en la carpeta `themes/` que hay junto a `config.json` (en Linux `~/.config/misoku/themes/`). Al copiar uno ahí, aparece en el selector sin reiniciar la app; si editas el archivo, el cambio se ve a los pocos segundos. Un tema propio con el mismo nombre que uno incluido no gana: los incluidos mandan.

`themes_dir` cambia esa carpeta por otra, y acepta una carpeta llena de JSON o un único archivo de paleta. Es el ajuste para llevar tus temas en un vault o en un repo aparte.

## Tu turno

1. Ejecuta `:colo` con un nombre, por ejemplo `:colo catppuccin`, y reinicia.
2. Copia un JSON de tema en la carpeta `themes/` y míralo aparecer en el selector sin reiniciar.
3. Cambia `themes_dir` a otra carpeta y comprueba que Misoku usa los temas de ahí.



Siguiente: [[E10 Carpetas en config.json]]
