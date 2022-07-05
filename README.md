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
Así mismo, cada tracker mantiene una salva de sus datos para poder recuperarse en el último estado conocido ante una posible caída del servidor o cualquier otra eventualidad que impida su correcta ejecución. 

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

### DHT(Kademlia)

En el protocolo de Bittorrent implementado, se utiliza una base de datos distribuida(**DHT**), basado en Kademlia([paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)). Esta base de datos guardará la ubicación de los peers de la red que poseen un determinado archivo. El objetivo de usar **Kademlia** es dotar al protocolo de una alternativa al **tracker**, otra forma de evitar tener en estos un punto de falla única. De esta forma a través del **dht** un cliente puede obtener la ruta de los peers que poseen el archivo que se quiere descargar.

La idea es que cada **peer** de la red tenga un nodo de la base de datos y mantener la información mencionada distribuida entre estos nodos, de forma que se explote al máximo la capacidad de la red. Con esto se busca evitar dos posibles problemas, el primero consiste en que pueda existir un archivo **.torrent** que no especifique ningún **tracker** que contenga información sobre él(**trackerless torrent**) y el segundo es que los **tracker** con la información sobre el torrent no respondan a las consultas.

#### - Estructura de la base de datos:

A cada nodo de **kademlia** se le asocia un **ID** random en el mismo espacio de 20 bytes que el **infohash** de los archivos **.torrent**.

A su vez, se asocia una dirección **IP** y un puerto(**port**). La primera coincide con el **IP** del peer que lo contiene. Y el segundo será el puerto a través del cuál se comunicará con los demás nodos del **dht**. Las conecciones que se utilizan son **udp**, factible para el protocolo debido a la baja latencia que estas garantizan, lo que permite hacer consultas de forma muy rápida a varios nodos, algo fundamental en kademlia. Además, debido a la alta capacidad de replicación del protocolo, no le afecta la incapacidad de las conecciones **udp** de garantizar la integridad de los paquetes que se envían, además de que en el protocolo no se retransmiten paquetes.

Una métrica de distancia se utiliza para determinar la "cercanía" entre dos nodos del **dht** o un nodo y el **infohash** de un torrent. La métrica que se utiliza es el **XOR** entre los bits del id o el infohash en cada caso. Se interpreta como un entero sin signo.

$$dist(A,B)\ =\ dist(B,A)\ =\ |A\ xor\ B|$$

Entre las ventajas de utilizar esta métrica está el hecho de que es simétrica, o sea, si el nodo **A** es "cercano" al nodo **B**, entonces el nodo **B** también es "cercano" a **A**, lo que en cuestiones del protocolo hace igualmente rápida la comunicación en ambos sentidos. Esto es una de las ventajas que **Kademlia** ofrece respecto a otras alternativas como **CHOORD**.

De forma general, cada nodo mantendrá una tabla de rutas, donde mantendrá la información de los nodos del **dht** de los que tiene conocimiento, más adelante se expilcará mejor esta estructura.

Además cada nodo mantiene un almacenamiento(**Store**), en el que mantiene las rutas de los peers donde descargar un archivo determinado, dado el **infohash** del torrent.

Luego, el cliente para encontrar las rutas de los peers que han descargado el archivo al que corresponde un torrent determinado, buscarán usando el **dht** sucesivamente en los nodos más próximos al **infohash** del torrent, hasta encontrar uno donde se tenga la información.

Para implementar el **dht** se crea el módulo **dht**. Este módulo permite ejecutar una instancia de nodo del **dht**, a través de la estructura **DHT**.


#### -Routing Table:

La tabla de rutas de cada nodo cubre el espacio de 0 a 2^160, correspondiente a todos los posibles id de los nodos del dht. Esta tabla constituye punto de partida para las consultas a otros nodos, pues solo se pueden enviar requests a nodos de los que se conozca su dirección. Así mismo, la respuesta a varias de las posibles consultas del protocolo se encuentran en esta tabla.

Solo se mantendrán en la tabla nodos **buenos**, o sea nodos que hayan respondido a consultas en un tiempo definido a la hora de crear la instancia del dht, en principio 15 minutos. El parámetro con el que se regula este tiempo es **TimeToRefreshBuckets**. 

La tabla de rutas se subdivide en 160 **buckets**. En el bucket **i** se guarda la información correspondiente a los nodos cuyo id difiere con el del nodo correspondiente a la tabla a partir del **i-ésimo** bit. De esta forma, a medida que crece **i**, los nodos del **bucket** se hacen más "cercanos" al nodo actual, teniendo en cuenta la noción definida de distancia. Si se analiza esta forma de organizar los buckets, puede verse que, del nodo **i** al nodo **i+1**, se reduce a la mitad la cantidad de nodos que tentativamente pudieran caer en el bucket, lo que hace q a medida que crece **i**, sea más exhaustivo el contenido de cada **bucket**, hasta el último, donde solo habrá un nodo, que será el más cercano al actual. Sobre este prinicipio se sustenta la búsqueda en **kademlia**. Cada uno de estos **buckets** mantiene una propiedad **lastTimeChanged** correspondiente al último momento en que se actualizó su contenido, de forma que si este tiempo se hace mayor que **TimeToRefreshBuckets**, se trata de actualizar su contenido, tomando un id random en el espacio correspondiente al bucket y realizando con sultas en el dht a partir de este id(más adelante se explica mejor).

En la implementación que se brinda, la tabla de ruta se representa a través de la estructura **RoutingTable**, que contiene la información referente al **ID**, **IP** y **puerto**, del nodo del dht, un slice de **kBuckets**(estructura utilizada para representar cada bucket), que se inicializa con tamaño 160 y un canal **lock**, que se utiliza para controlar el acceso concurrente a la tabla de rutas.

#### -KStore

Para el almacenamiento de cada nodo del dht se define la estructura **Kstore**. Esta contiene un campo **data** que será un diccionario con llave y valor de tipo **string**. La llave corresponde al **infohash** de los torrent de los que se ha obtenido información y el valor contiene las rutas a los peers que contienen el arcivo especificado. Por cada **infohash** y **ruta** se mantiene un tiempo de expiración y un tiempo de republicación en los diccionarios **mapExpirationTime** y **mapTimeToRepublish**. Igualmente se mantiene un canal **lock** para controlar el acceso concurrente al almacenamiento.

Con el tiempo de expiración se garantiza no tener almacenado en el dht información obsoleta, por lo que se mantiene el chequeo sistemático de este parámetro. Igual para el caso del tiempo de republicación, se utiliza para mantener el dht con información actualizada, de forma que cada nodo que mantiene una información la republica en la red cada en cada intervalo de tiempo definido por el parámetro. Ambos procesos se realizan utilizando go routines que se mantienen en background chequeando estos parámetros, una vez inicializado el **dht**.

#### - Mensajes del protocolo de kademlia:

La comunicación entre los nodos del dht se hace utilizando mensajes **RPC**, a través de puertos **udp** como se explicó anteriormente. Hay tres tipos de mensajes definidos en el protocolo, **QueryMessages**, **ResponseMessages** y **ErrorMessages**. Estos mensajes se envían en forma de diccionarios **bencode**, para ello se utiliza el paquete de golang [bencode-go](https://pkg.go.dev/github.com/jackpal/bencode-go@v1.0.0).

Las posibles consultas del protocolo son 4:

1. **Ping**: Este mensaje es el más simple del protocolo y por lo general se utiliza para chequear si un nodo del que se tiene conocimiento está activo.

2. **FindNode**: Se utiliza para pedir a un nodo los **k** nodos más cercanos a un **id** que se envía como parámetro en la consulta. De forma que la respuesta será la información compacta(CompactInfo) de estos k nodos. Este dato contiene los bytes correspondientes al id, ip y puerto del nodo, en ese orden, y se envía como **string**.

3. **GetPeers**: Esta consulta envía como parámetro el **infohash** de un torrent específico, de forma que si el nodo consultado tiene la información referente a los peers que contienen el archivo que se busca, retorna las rutas correspondientes, si no se comporta como **FindNode**, devolviendo los k nodos más cercanos al infohash, para continuar la búsqueda a partir de ellos.

4. **AnnouncePeer**: Este mensaje se utiliza en el protocolo para anunciar que el peer que contiene el nodo que envía la consulta está descargando el archivo correspondiente al torrent cuyo **infohash** se pasa como parámetro de la consulta. En este mensaje se especifica el puerto por el que se puede descargar el archivo.

Cada nodo mantendrá abierta una conexión **udp**, por el puerto definido para el nodo, por la que se mantendrá escuchando los paquetes que le envían, que pueden ser tanto consultas como respuestas a consultas previas. A través de esta misma conexión, que se mantendrá como una propiedad del nodo, se envíaran los mensajes desde este nodo a cualquier otro del dht.

#### - LookUP:

Quizás la funcionalidad más importante del protocolo de kademlia es el **LookUP**. Esta función recibe como parámetro el tipo de consulta a realizar, "find_node" o "get_peers" y el id o infohash que se pasará como parametro. La idea de esta función es, en el caso del "find_node" buscar iterativamente en la red los k nodos más cercanos a un id recibido como parámetro. Para ello primero se envían FindNode request a los **ALPHA**(normalmente se fija en 3) nodos más cercanos al id, de los que se tiene conocimiento en la tabla de rutas, luego los nodos que se obtengan se agregan a una estructura que los mantiene ordenados según la distancia al id recibido y se le vuelven a enviar el mismo mensaje FindNode a los ahora más cercanos, obtenidos en respuesta a request previos, este proceso se hace hasta que no se obtenga ningún nodo que mejore la distancia al id. En el caso de "get_peers" se hace el mismo proceso, pero se para cuando se obtenga una respuesta válida, o sea información sobre peers que contienen el infohash.

Esta función es la base de las principales funcionalidades del protocolo **Kademlia**. Para encontrar los peers que contienen un determinado infohash se hace **LookUP** pasando como parámetro el infohash deseado. Igualmente para anunciar en el dht que se está descargando el archivo correspondiente a un infohash, primero se hace **LookUP**, pasando como parámetro "find_node" y el infohash del archivo que se está descargando, y se obtienen los k nodos más cercanos al infohas, a los que se les enviará un request del tipo **AnnouncePeer**. Precisamente este es el mecanismo que permite que el dht implementado con kademlia tenga mucha repliación. También se utiliza el LookUP para mantener la tabla de rutas actualizada.

#### - JoinNetwork:

Cuando un nuevo nodo del dht se crea, se necesita conocer al menos la ruta de un nodo activo del dht, comunmente a estos nodos se les llama **bootstraps nodes**. Estos nodos se utilizarán como entrada a la red.

Para unirse a la red, se le hace ping a la ruta de un nodo **bostrap** y luego, si responde, se hace **LookUP**, pasando como parámetro "find_node" y el propio id del nodo, para así obtener respuestas de muchos nodos del dht y actualizar la tabla de rutas.

#### - Crear una instancia de nodo de **DHT**:

Una vez importado el módulo de **Golang** **dht** para crear una instancia de un nodo, primero deben definirse las opciones para configurar el nodo:

```
   options := &dht.Options{
		ID:                   id,
		IP:                   ip,
		Port:                 port,
		ExpirationTime:       time.Duration,
		RepublishTime:        time.Duration,
		TimeToDie:            time.Duration,
		TimeToRefreshBuckets: time.Duration,
	}
```
Luego debe obtenerse la instancia del nodo y crear la conexión **udp** utilizada para la comunicación en el dht, para esto se utiliza la función RunServer:

```
   dhtNode := dht.NewDHT(options)
   exitChan := make(chan string)
	go dhtNode.RunServer(exitChan)
```

Para lograr que se una a la red el nodo se ejecuta la función JoinNetwork:

```
   dhtNode.JoinNetwork(addrBootstrap)
```

Obtener los peers que contienen un infohash:

```
   peersAddr, err := dhtNode.GetPeersToDownload(infohash)
```

Anunciar en el dht que se está descargando el archivo correspondiente a un infohash por un puerto **port**:

```
   dht1.AnnouncePeer(infohash, port)
```




