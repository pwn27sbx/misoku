# A06 · Historia de la nota

**Vas a aprender:** a ver qué cambió en una nota, comparar versiones y devolver el texto antiguo al editor sin perder lo de ahora.

`:history` pide al repositorio git del vault los commits que tocaron esta nota, renombrados incluidos, y los lista con su fecha al lado. No es un historial propio de Misoku: es el de git, así que el vault tiene que ser un repositorio con commits, como el que crea el sync.

Al moverte por la lista con `j` y `k`, o con la rueda, el panel de al lado enseña esa versión. Es una lectura, no un cambio: la nota sigue como está.

`Enter` sobre una versión mete su texto en el editor como un solo cambio deshacible. Nada se escribe hasta que guardas, y si te arrepientes, `u` lo revierte de una vez. En una nota sin commits, o en un vault sin git, no hay diálogo: la barra dice por qué y no pasa nada más. En el vault del tutorial no hay git, así que ese aviso es lo que verás; los pasos de restaurar se practican en un vault con sync.

## Tu turno

1. En un vault con commits, abre una nota que haya cambiado y ejecuta `:history`.
2. Muévete con `j` y `k` (o con la rueda) y mira la versión que aparece al lado.
3. Añade `presupuesto: 900` al front matter, guarda, cámbialo por `presupuesto: 950`, guarda y, cuando haya commits, pulsa `Enter` en la versión vieja y deshaz con `u`.
4. Ejecuta `:history` en una nota sin commits y lee el aviso de la barra.



Siguiente: [[A07 Menciones y backlinks]]
