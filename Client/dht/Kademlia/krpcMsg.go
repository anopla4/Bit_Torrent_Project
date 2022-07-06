package dht

import (
	"crypto/rand"
	"strconv"
)

type message struct {
	TransactionID string `bencode:"t"`
	TypeOfMessage string `bencode:"y"`
}

type krpcErroInt interface {
	ErrorKRPC() string
}

// krpcError struct to error handling in kademlia protocol
type krpcError struct {
	krpcErroInt
	message
	ErrorField []string `bencode:"e"`
}

func (err *krpcError) ErrorKRPC() string {
	return err.ErrorField[1]
}

// GenericError generic error type
type GenericError struct {
	krpcError
}

func newGenericError(errM string) *GenericError {
	tid, err := newTransactionID()
	if err != nil {
		return nil
	}
	return &GenericError{
		krpcError: krpcError{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "e",
			},
			ErrorField: []string{"201", errM},
		},
	}
}

// ServerError server error type
type ServerError struct {
	krpcError
}

func newServerError(errM string) *ServerError {
	tid, err := newTransactionID()
	if err != nil {
		return nil
	}
	return &ServerError{
		krpcError: krpcError{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "e",
			},
			ErrorField: []string{"202", errM},
		},
	}
}

// ProtocolError protocol error type(bad token, invalid arguments...)
type ProtocolError struct {
	krpcError
}

func newProtocolError(errM string) *ProtocolError {
	tid, err := newTransactionID()
	if err != nil {
		return nil
	}
	return &ProtocolError{
		krpcError: krpcError{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "e",
			},
			ErrorField: []string{"203", errM},
		},
	}
}

// MethodUnknownError generic error type
type MethodUnknownError struct {
	krpcError
}

func newMethodUnknownError(errM string) *MethodUnknownError {
	tid, err := newTransactionID()
	if err != nil {
		return nil
	}
	return &MethodUnknownError{
		krpcError: krpcError{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "e",
			},
			ErrorField: []string{"204", errM},
		},
	}
}

// QueryMessage struct for query message in kademlia protocol
type QueryMessage struct {
	message
	QueryName string            `bencode:"q"`
	Arguments map[string]string `bencode:"a"`
}

func newQueryMessage(queryName string, args map[string]string) (*QueryMessage, krpcErroInt) {
	if queryName == "ping" {
		if len(args) != 1 {
			return nil, newProtocolError("one argument required for ping request and " + strconv.Itoa(len(args)) + "were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("invalid argument for ping message")
		}
		tid, err := newTransactionID()
		if err != nil {
			return nil, newProtocolError("error while getting transactionID")
		}
		return &QueryMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "q",
			},
			QueryName: queryName,
			Arguments: args,
		}, nil

	}
	if queryName == "get_peers" {
		if len(args) != 2 {
			return nil, newProtocolError("two arguments required for get_peers request and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for get_peers request")
		}
		_, in = args["info_hash"]
		if !in {
			return nil, newProtocolError("info_hash argument required for find_node request")
		}
		tid, err := newTransactionID()
		if err != nil {
			return nil, newGenericError("error while getting transactionID")
		}
		return &QueryMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "q",
			},
			QueryName: queryName,
			Arguments: args,
		}, nil

	}
	if queryName == "find_node" {
		if len(args) != 2 {
			return nil, newProtocolError("two arguments required for find_node request and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for find_node request")
		}
		_, in = args["target"]
		if !in {
			return nil, newProtocolError("target argument required for find_node request")
		}
		tid, err := newTransactionID()
		if err != nil {
			return nil, newGenericError("error while getting transactionID")
		}
		return &QueryMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "q",
			},
			QueryName: queryName,
			Arguments: args,
		}, nil

	}
	if queryName == "announce_peer" {
		if len(args) != 3 {
			return nil, newProtocolError("four arguments required for announce_peer request and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for announce_peer request")
		}
		_, in = args["info_hash"]
		if !in {
			return nil, newProtocolError("info_hash argument required for announce_peer request")
		}
		// _, in = args["token"]
		// if !in {
		// 	return nil, newProtocolError("token argument required for announce_peer request")
		// }
		_, in = args["port"]
		if !in {
			return nil, newProtocolError("port argument required for announce_peer request")
		}
		tid, err := newTransactionID()
		if err != nil {
			return nil, newGenericError("error while getting transactionID")
		}
		return &QueryMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "q",
			},
			QueryName: queryName,
			Arguments: args,
		}, nil

	}
	return nil, newMethodUnknownError("method name not valid")
}

// ResponseMessage struct for response message in kademlia protocol
type ResponseMessage struct {
	message
	Response map[string]string `bencode:"r"`
}

func newResponseMessage(queryName string, tid string, args map[string]string) (*ResponseMessage, krpcErroInt) {
	if queryName == "ping" {
		if len(args) != 1 {
			return nil, newProtocolError("one argument required for ping Response and " + strconv.Itoa(len(args)) + "were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("invalid argument for ping message")
		}

		return &ResponseMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "r",
			},
			Response: args,
		}, nil

	}
	if queryName == "get_peers" {
		if len(args) != 2 {
			return nil, newProtocolError("two arguments required for get_peers response and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for get_peers response")
		}
		// _, in = args["token"]
		// if !in {
		// 	return nil, newProtocolError("token argument required for find_node response")
		// }
		_, inPeer := args["values"]
		_, inNodes := args["nodes"]
		if !inPeer && !inNodes {
			return nil, newProtocolError("one of values or nodes arguments are required for get_peers response")
		}
		if inPeer && inNodes {
			return nil, newProtocolError("values and nodes arguments can not be both in get_peers response")
		}
		return &ResponseMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "r",
			},
			Response: args,
		}, nil

	}
	if queryName == "find_node" {
		if len(args) != 2 {
			return nil, newProtocolError("two arguments required for find_node response and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for find_node response")
		}
		_, in = args["nodes"]
		if !in {
			return nil, newProtocolError("nodes argument required for find_node response")
		}
		return &ResponseMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "r",
			},
			Response: args,
		}, nil

	}
	if queryName == "announce_peer" {
		if len(args) != 1 {
			return nil, newProtocolError("one argument required for announce_peer response and " + strconv.Itoa(len(args)) + " were given")
		}
		_, in := args["id"]
		if !in {
			return nil, newProtocolError("id argument required for announce_peer response")
		}
		return &ResponseMessage{
			message: message{
				TransactionID: tid,
				TypeOfMessage: "r",
			},
			Response: args,
		}, nil

	}
	return nil, newMethodUnknownError("method name not valid")
}

func newTransactionID() (string, error) {
	tid := make([]byte, 1)
	_, err := rand.Read(tid)
	if err != nil {
		return "", err
	}
	return string(tid), nil
}
