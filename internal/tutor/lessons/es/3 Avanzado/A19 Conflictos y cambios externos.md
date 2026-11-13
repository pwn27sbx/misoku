# A19 · Conflictos y cambios externos

**Vas a aprender:** qué pasa cuando el archivo cambia por fuera mientras lo tienes abierto, y cómo Misoku evita perder texto.

Si guardas una nota cuyo archivo cambió en el disco, las dos versiones valen. Misoku no elige: escribe el texto del disco en una copia de conflicto al lado, con la fecha en el nombre, y deja el tuyo en la nota. La barra lo dice con `Se conservaron las dos:` y el nombre de la copia, para que sepas dónde quedó lo otro.

Lo mismo vale para una nota que borras o renombras por fuera. La pestaña sigue con su texto, avisa una vez y, si intentas guardar, pregunta antes de volver a crear el archivo. Nada se recrea en silencio.

El sync usa la misma idea entre dos copias del vault: cuando las dos tocan las mismas líneas, la copia de conflicto guarda una versión y la nota la otra, y tú decides cuál se queda. Dos textos, dos archivos, ninguna pérdida.

## Tu turno

1. Con esta nota abierta, cámbiala desde fuera (un editor o un `echo`) y guarda aquí: mira `Se conservaron las dos:` en la barra y la copia de conflicto junto a la nota.
2. Borra o renombra una nota por fuera, vuelve a Misoku y comprueba que avisa una vez y que al guardar pregunta antes de recrearla.
3. Con el sync montado en dos copias del vault, provoca un conflicto en la misma línea y mira la copia `plan (conflict …).md`.
4. Comprueba que tu texto está en la nota y el otro en la copia.



Siguiente: [[A20 Editar carpetas en el editor]]
