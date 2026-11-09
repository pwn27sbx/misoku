# A10 · Enlaces a archivos y enlaces peligrosos

**Vas a aprender:** a enlazar un archivo del vault y a reconocer cuándo un enlace merece una pregunta antes de abrirse.

Un enlace de Markdown normal puede apuntar a cualquier archivo, no solo a una nota: `[menú](menu.pdf)` se lo entrega a tu sistema para que lo abra con el programa que corresponda. Es la forma de tener un PDF, una hoja de cálculo o una imagen a un clic de la nota.

Cuando el destino no es tan inocente, Misoku pregunta. Un esquema como `file:///`, una ruta que se sale del vault o un archivo ejecutable abren un diálogo con Cancelar ya elegido, para que un clic de más no lance nada.

El diálogo es la regla: si el destino no está claro, primero se pregunta y la respuesta segura es la que ya viene marcada. Aceptar es siempre una decisión, no un descuido.

## Tu turno

1. Escribe `[menú](menu.pdf)`, o el enlace a un archivo que ya esté en el vault, y síguelo: se abre con el programa del sistema.
2. Escribe `[raro](file:///etc/hosts)`, intenta seguir el enlace y comprueba que el diálogo pide permiso con Cancelar elegido.



Siguiente: [[A11 Grafo local]]
