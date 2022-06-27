package dht

import "crypto/rand"

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
	QueryName string                 `bencode:"q"`
	Arguments map[string]interface{} `bencode:"a"`
}

func newQueryMessage(queryName string, args map[string]interface{}) (*QueryMessage, krpcErroInt) {
	if queryName == "ping" {
		if len(args) != 1 {
			return nil, newProtocolError("one argument required for ping request and " + string(len(args)) + "were given")
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
			return nil, newProtocolError("two arguments required for get_peers request and " + string(len(args)) + " were given")
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
			return nil, newProtocolError("two arguments required for find_node request and " + string(len(args)) + " were given")
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
	return nil, newMethodUnknownError("method name not valid")
}

// ResponseMessage struct for response message in kademlia protocol
type ResponseMessage struct {
	message
	Response map[string]interface{} `bencode:"r"`
}

//TODO newResponceMessage

func newTransactionID() (string, error) {
	tid := make([]byte, 1)
	_, err := rand.Read(tid)
	if err != nil {
		return "", err
	}
	return string(tid), nil
}
