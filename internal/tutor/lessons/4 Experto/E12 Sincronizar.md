# E12 · Sincronizar

**Vas a aprender:** el sync a mano, al abrir y por tiempo.

`space s` o `:sync` lanza un sync completo: guarda lo pendiente en un commit, trae lo del remoto, mezcla y empuja. Mientras corre, la barra lo dice; al terminar, `ok` o el motivo por el que se paró. Es el botón que vas a pulsar cuando quieras asegurar tu trabajo antes de cerrar el portátil.

Hay dos syncs automáticos más. Uno al abrir un vault de la lista: si entras a tus notas, Misoku sincroniza una vez sin preguntar. Y otro por tiempo, cada cinco minutos por defecto, mientras la app está abierta. Ese intervalo se cambia por vault con `interval`, con un mínimo de un minuto; `off` lo apaga y deja los otros dos.

Los tres hacen lo mismo por dentro. La diferencia es quién los dispara: tú, la apertura o el reloj.

## Tu turno

1. Pulsa `space s` (o `:sync`) y espera el final.
2. Vuelve a abrir un vault de la lista y mira el sync de la apertura.
3. Espera el sync periódico, o baja `interval` para no esperar cinco minutos.



Siguiente: [[E13 Qué hace el sync]]
