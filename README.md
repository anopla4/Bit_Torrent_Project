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

## Detalles de implemetación 

### Tracker

La red bitTorrent implementada cuenta con un protocolo de comunicación a través de trackers. Los trackers son servidores que tienen la funcionalidad de facilitar la comunicación entre peers dentro de la red bitTtorrent. Cada .torrent puede contar entre sus campos con un Announce y/o Announce List para proveer un tracker o una lista trackers donde puede tenerse informaciones acerca del mismo. 

Los trackers no almacenan los files ni los .torrents sino que tienen información acerca de los mismos como: cantidad de descargas, peers con el torrent completo o incompleto. La comunicación entre los peers y el tracker ocurre a través de 3 servicios proveídos por este último.:

1. Publish: El peer hace conocer al tracker que posee completamente el file especificado en el .torrent de forma tal que pasa a formar parte del conjunto de seeders para dicho torrent. 

2. Announce: Es el proceso mediante el cual el peer solicita un peer-set(conjunto de peers poseedores total o parcialmente de un determinado file) para iniciar la descarga del file especificado por el .torrent. Así mismo el peer comunica cuando se detiene el proceso de descarga, cuando se inicia, y cuando se completa para cambiar su estado dentro de la red. 

3. Scrape: Mediante esta comunicación un peer puede solicitar al tracker información sobre un conjunto de torrents. La información proporcionada por el tracker incluye para cada torrent: Número de descargas hechas, número de peers en estado de el descarga incompleto, y número de peers en estado de descarga completo. 

La implementación del tracker se realizó en base a microservicios, con lo cual se eligió gRPC para proveer los mismos. Se eligió gRPC debido a que provee una capa adicional de seguridad así como estar más directamente orientada a servicios y a entornos donde el número de comunicaciones puede ser elevado.  Además la transmisión de información a través de protocols buffers es más ligera que la realizada por Rest/http1.1 a través de jsons. El manejo de conexiones de Go, su orientación a una concurrencia y paralelismo eficiente y su rapidez de ejecución lo convirtieron en el lenguaje elegido a la hora de implementar el servidor tracker. A su vez las conexiones fueron aseguradas a través de TLS. 

Para mitigar el hecho de que el tracker constituye un servidor centralizado y por tanto un punto de falla dentro de una red propiamente distribuida, se implementó un protocolo de comunicación entre trackers, de tal forma que ocurra un proceso de replicación parcial de la información a través de distintos tracker conocidos entre sí. 

La comunicación entre tracker cuenta con dos servicios:

1. KnowMe: Es el mensaje a través del cual un tracker intenta darse a conocer a un posible conjunto de otros trackers proveídos a él por un administrador de la red. Para que la comunicación KnowMe sea efectiva, la clave de red(redKey) hasheada debe coincidir entre ambos trackers. Si la comunicación es aprobada por el tracker receptor, ambos(solicitante y receptor) pasan a ser incluidos en la lista de replicación del otro respectivamente. Todo tracker que pertenece a la lista de replicación de otro tracker recibirá un RePublish por cada Publish o complete Announce recibido por el otro. 

2. RePublish: es un mensaje entre trackers conocidos donde el receptor de un Publish o complete Announce por parte de un peer envía los datos de ese .torrent y ese peer a otro tracker. De forma tal que este último contará también con la información de que dicho peer es un seeder de ese .torrent. De esta forma ante la eventual caída de un tracker, el otro puede ser capaz de proveer respuesta a los clientes acerca de la mayoría de torrents conocidos por el primero. 

La comunicación entre trackers también está implementada sobre gRPC. 
Así mismo, cada tracker mantiene un salva de sus datos para poder recuperarse en el último estado conocido ante una posible caída del servidor o cualquier otra eventualidad que impida su correcta ejecución. 

Para levantar un servidor tracker:

   tracker/server> go run server.go .... 

La funcionalidad del tracker está parametrizada, por lo cual, recibe los siguientes flags:

-tls : añadir seguridad TLS a la comunicación 

-redKey <password>(requerido) : crea el hash de comunicación de reconocimiento entre trackers
  
-ip <ip>  : Define la ip donde se intentará levantar el servidor. Default localhost
  
-port <port> : Define el puerto. Default 8168
  
-load : Cargar datos almacenados al momento de levantar el servidor
  
-saveTime <t>: Tiempo t en segundos entre cada salva. Default 600
  
-nBkTk <json_path>: Si aparece, define la ruta de un archivo .json que contiene las direcciones de los trackers a los que debe intentar darse a conocer. 

     Estructura del json:

       {
          "192.168.169.12:5030" :{
              "ip": "192.168.169.32",
              "port": 5030
           },
           .... 
       } 

