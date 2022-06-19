package uploader_client

import (
	"context"
	"io"
	"log"
	"net"
	"torrent_peerpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	torrent_peerpb.UnimplementedDownloadServiceServer
}

func (*Server) SendMessage(stream torrent_peerpb.DownloadService_SendMessageServer) error {

	for {
		req, err := stream.Recv()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error while reading client stream: %v\n", err)
			return err
		}

		m_req := req.GetMessage()

		m_resp := &torrent_peerpb.Message{
			Length:  m_req.GetLength(),
			Id:      m_req.GetId(),
			Payload: m_req.GetPayload(),
		}

		sendErr := stream.Send(&torrent_peerpb.SendMessageResponse{
			Message: m_resp,
		})

		if sendErr != nil {
			log.Fatalf("Error while sending response to client: %v\n", err)
		}
	}

	return nil
}

func (*Server) Handshake(ctx context.Context, req *torrent_peerpb.HandshakeRequest) (*torrent_peerpb.HandshakeResponse, error) {
	length_pstr := req.GetLengthPstr()
	pstr := req.GetPstr()
	info_hash := req.GetInfoHash()
	peer_id := req.GetPeerId()

	res := &torrent_peerpb.HandshakeResponse{
		LengthPstr: length_pstr,
		Pstr:       pstr,
		InfoHash:   info_hash,
		PeerId:     peer_id,
	}

	return res, nil
}

func StartServer(port string) {
	// Start listening on port 50051
	lis, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	tls := false

	// SSL settings
	if tls {
		certFile := "ssl/server.crt"
		keyFile := "ssl/server.pem"
		creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)

		if sslErr != nil {
			log.Fatalf("Failed loading certificates: %v", sslErr)
			return
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create server and start serving
	s := grpc.NewServer(opts...)
	torrent_peerpb.RegisterDownloadServiceServer(s, &Server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
