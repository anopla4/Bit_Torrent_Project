# Bit_Torrent_Project

## Información
-Nombre del proyecto: Bit Torrent

-Reporte: https://www.overleaf.com/read/skndhhhwjnnm

## Integrantes
-Ana Paula Argüelles Terrón C-412 (@anoppa)

-Javier Alejandro Lopetegui González C-412 (@jlopetegui98)

-Abel Molina Sánchez C-411 (@ams_1927)

## Resumen del proyecto

FTP normalmente resulta una alternativa fácil a la hora de compartir y descargar archivos. Cuando
un archivo se elimina del FTP, para recuperar espacio, se pierde la oportunidad de obtenerlo. En
este caso el problema está en que el servicio se encuentra centralizado. FTP también es un estándar
de los más antiguos de internet en capa de aplicación. Existen alternativas descentralizadas, como
BitTorrent, que garantizan que mientras alguna parte del sistema tenga el archivo este va a estar
disponible para los demás clientes de la red. Esto se podría realizar utilizando el almacenamiento de
los propios clientes como almacenamiento del sistema.

Para la confección de este sistema se requiere de implementar dos elementos, posteriormente
descritos: Cliente y Tracker.
### Cliente
Un cliente es un nodo de la red que sirve y descarga archivos. Un archivo en descarga o descargado
debe estar disponible para el resto de los clientes. El alcenamiento solo puede estar del lado de los
clientes, no se puede utilizar un servicio externo para esta tarea. Se debe proveer una aplicación
gráfica para una mejor interacción con el sistema.
### Tracker
Un tracker es el elemento en el sistema que mantiene actualizado qué clientes poseen qué archivo.
La información se publica y descarga a través de archivos torrent. Esto quiere decir que si un cliente
quiere servir determinado archivo (o grupo de archivos) debe crear un archivo torrent que así lo
declare y publicarlo en el tracker. Este elemento no puede representar un punto de falla única en su
sistema.
