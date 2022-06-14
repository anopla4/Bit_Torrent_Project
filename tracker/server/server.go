package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/examples/data"

	"github.com/golang/protobuf/proto"

	pb "github.com/anoppa/Bit_Torrent_Project/tracker/trackerpb"
)

// AnnounceQuery es una representacion de una announce get query
type AnnounceQuery struct {
	InfoHash   string //dic bencodificado
	PeerID     string // Id del peer
	IP         net.IP //Ip del peer Opcional
	Port       int16  // Port del per Opcional
	Uploaded   int
	Downloaded int
	Left       int
	Event      int
	NumWant    int //Cantidad de peer que solicita el peer, Default 50
}

// AnnounceResponse es una representacion de una respuesta a un announce get query
type AnnounceResponse struct {
	Interval   int
	TrackerID  int
	Complete   int
	Incomplete int
	Peers      string
}

type TrackerServer struct{
	AnnReq  AnnounceQuery
	AnnResp AnnounceResponse
}


func (tk *TrackerServer) Announce(ctx context.Context, annq *pb.AnnounceQuery) (*pb.AnnounceResponse, error){
	
}