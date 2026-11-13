# A08 · Renombrar sin romper enlaces

**Vas a aprender:** a renombrar una nota sin dejar enlaces rotos y a enlazar la nota correcta cuando dos se llaman igual.

Renombrar en el explorador con `r` no es solo cambiar el archivo. Misoku mira qué notas enlazan a esta y te pregunta `¿Actualizar N enlaces en M notas?`. Si aceptas, reescribe esos enlaces en todas ellas, en un paso, y no queda ninguno apuntando a un nombre que ya no existe. Si dices que no, el archivo cambia y los enlaces se quedan como estaban.

Los nombres repetidos son el otro lado. Con dos notas llamadas `Maleta`, una en `Viajes` y otra en `Casa`, el enlace corto sería ambiguo. Por eso, cuando eliges la de una carpeta, la lista de sugerencias escribe la ruta: el enlace lleva la carpeta dentro y señala exactamente esa nota.

Las dos reglas juntas son la razón de que enlazar con corchetes salga barato: la app mantiene los nombres al día y desambigua cuando hace falta.

## Tu turno

1. Crea una nota llamada `Maleta`, escribe `[[Maleta]]` en esta nota, renómbrala con `r` a `Equipaje` y acepta actualizar los enlaces.
2. Crea `Casa` y `Viajes` con una nota `Maleta` dentro de cada una, enlaza la de `Casa` y mira que el enlace lleva la carpeta en la ruta.



Siguiente: [[A09 Incrustar]]
