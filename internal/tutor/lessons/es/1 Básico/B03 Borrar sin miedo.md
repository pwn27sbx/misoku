# B03 · Borrar sin miedo

**Vas a aprender:** a borrar un carácter con `x` y una línea entera con `dd`.

`x` borra la letra que tiene el cursor, sin entrar en INSERT. `dd` es tu primer operador: `d` es "borrar" y la segunda `d` dice "la línea". Vim los combina, y esa idea (operador más objeto) es la que usarás toda la vida: `dw` borra una palabra y `d$` borra hasta el final de la línea, aunque hoy solo necesitas `dd`.

## Tu turno

1. En la zona de pruebas, pon el cursor sobre la `e` de `hospedaje` y pulsa `x`.
2. Pon el cursor en la línea `borrador` y pulsa `dd`: la línea desaparece entera.

## Tu zona de pruebas

hospedaje

borrador

Siguiente: [[B04 Moverte sin ratón]]
