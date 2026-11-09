# E11 · Sync en config.json

**Vas a aprender:** qué carpetas sincronizan y qué recuerda Misoku de ti.

`sync.vaults` es la lista de vaults que Misoku sincroniza. Una carpeta que no está en la lista no ejecuta git nunca: ni al guardar, ni al abrir, ni al salir. Por eso el sync es opt-in y por eso añadir una entrada es el paso que lo activa; cada entrada puede llevar su `interval` y su `device`, el nombre que firma los commits y las copias de conflicto.

La forma de añadirla a mano es una entrada más del objeto, con la ruta como clave: `{"sync": {"vaults": {"~/Notas": {}}}}`. La ruta tiene que ser absoluta o empezar por `~`, igual que `notes_dir`. La forma cómoda es `:sync setup`, que la escribe por ti (lección E17).

`learn_offered` es otra clave pequeña: guarda `true` cuando ya respondiste al diálogo de primera vez, sea "Soy nuevo", "Ya tengo experiencia" o "Ahora no". Así no vuelve a salir en esa máquina; si borras la clave, vuelve a preguntar una vez.

## Tu turno

1. Añade un vault a `sync.vaults` y comprueba que solo ese sincroniza.
2. Mira `learn_offered` después de responder al diálogo de primera vez.



Siguiente: [[E12 Sincronizar]]
