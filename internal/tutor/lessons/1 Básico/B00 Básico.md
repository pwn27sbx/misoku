# Ruta Básico

Bienvenido al tutorial de Misoku. Esta ruta no asume nada: empiezas con la primera lección abierta y escribes en los primeros treinta segundos.

Cada ruta vive en su carpeta en el explorador (`1 Básico`, `2 Intermedio`, `3 Avanzado`, `4 Experto`), así que sus lecciones quedan juntas y en el orden en que se aprenden. `Después del tutorial.md` y `mapa.png` están en la raíz del tutorial.

Si llegaste desde el diálogo de primera vez, esto es lo que elegiste. **Soy nuevo** abre esta ruta; **Ya tengo experiencia** te lleva directo a Intermedio; **Ahora no** te deja en tus notas. Ese diálogo solo decide por dónde entras, y puedes cambiar de ruta cuando quieras desde la entrada **Learn** del inicio.

## Cómo se juega

Cada lección es una nota como esta. Arriba te explica qué vas a aprender y por qué; debajo, en **Tu turno**, están las tareas numeradas. Hazlas en la propia nota: Misoku mira lo que escribes o la acción que haces y te lo dice en la barra de estado.

La barra muestra la lección y el paso que toca, con lo que falta (`B03 · Paso 2/4 · …`). Cuando un paso se cumple ves `✓ Paso 2`; al terminar la lección, `✓ B03 superado` con el nombre de la siguiente; y al terminar la ruta, `🏆 Ruta Básico completa`.

Los ejemplos van siempre entre acentos graves: `i`, `ctrl+s`, `[[Nota]]`. El corrector ignora todo lo que está en código, así que copiar un ejemplo no cuenta; escríbelo tú en una línea nueva.

## Los dos modos

Misoku usa los modos de vim. En NORMAL cada letra es un comando; para escribir entras en INSERT con `i` (o `a`, `o`…) y sales con `Esc`. Si algo no responde como esperas, mira el modo: casi siempre es que sigues en INSERT.

## Las 26 lecciones

1. [[B01 Escribe en 30 segundos]] — `i`, `Esc` y `ctrl+s`
2. [[B02 Insertar donde quieras]] — `a`, `A`, `I`, `o`, `O`
3. [[B03 Borrar sin miedo]] — `x` y `dd`
4. [[B04 Moverte sin ratón]] — `hjkl`, flechas y saltos
5. [[B05 Deshacer y rehacer]] — `u` y `ctrl+r`
6. [[B06 Guardar con nombre y cerrar]] — `:w` y `space x`
7. [[B07 Pestañas]] — `H`, `L`, `]`, `[`
8. [[B08 El explorador y el foco]] — `space e` y el foco
9. [[B09 Abrir y crear notas]] — el árbol y `n`
10. [[B10 Renombrar y borrar]] — `r` y `d`
11. [[B11 Plegar y espiar]] — `l`, `h`, `z`, `W`, `p`
12. [[B12 El menú barra]] — el menú que escribe Markdown
13. [[B13 Títulos]] — de `#` a `####`
14. [[B14 Listas]] — viñetas, números y tareas
15. [[B15 Casillas, marcar]] — tres formas de marcar
16. [[B16 Casillas, quitar y clic]] — quitar la casilla y clicar
17. [[B17 Énfasis]] — negrita, cursiva, código y tachado
18. [[B18 Citas y separador]] — `>` y `---`
19. [[B19 Tu primera nota enlazada]] — `[[`
20. [[B20 Seguir un enlace]] — `gd`, `gx`, `Enter` y el ratón
21. [[B21 Leer sin editar]] — `space p` y `space r`
22. [[B22 Buscar dentro de la nota]] — `/`, `n`, `N`
23. [[B23 Pegar y equivocarse]] — `ctrl+v` y los errores
24. [[B24 El inicio de Misoku]] — `space d`
25. [[B25 Salir y volver]] — `:q`, `:wq`, `:q!`
26. [[B26 Desafío Básico]] — una lista de compras real

Cuando quieras parar, `ctrl+c` cierra Misoku y el progreso queda guardado paso a paso: al volver a `misoku tutor` sigues donde lo dejaste. Para empezar de cero, `misoku tutor --reset` manda el tutorial a la papelera.

Siguiente: [[B01 Escribe en 30 segundos]]
