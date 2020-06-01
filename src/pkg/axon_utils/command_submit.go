package axon_utils

import (
	context "context"
	errors "errors"
	log "log"

	proto "github.com/golang/protobuf/proto"
	uuid "github.com/google/uuid"

	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
)

func SendCommand(commandType string, command proto.Message, clientConnection *ClientConnection) error {
	data, err := proto.Marshal(command)
	if err != nil {
		log.Printf("Server: Error while marshalling command: %v", commandType)
		return errors.New("marshalling error")
	}
	serializedCommand := axon_server.SerializedObject{
		Type: commandType,
		Data: data,
	}

	return submitCommand(&serializedCommand, clientConnection)
}

func submitCommand(message *axon_server.SerializedObject, clientConnection *ClientConnection) error {
	conn := clientConnection.Connection
	log.Printf("Submit command: %v: %v", message.Type, conn)
	client := axon_server.NewCommandServiceClient(conn)
	log.Printf("Submit command: Client: %v", client)

	clientInfo := clientConnection.ClientInfo
	id := uuid.New()
	command := axon_server.Command{
		MessageIdentifier: id.String(),
		Name:              (*message).Type,
		Payload:           message,
		ClientId:          clientInfo.ClientId,
		ComponentName:     clientInfo.ComponentName,
	}
	log.Printf("Submit command: Command: %v", command)

	response, e := client.Dispatch(context.Background(), &command)
	log.Printf("Submit command: Response: %v: %v", response, e)
	if e != nil {
		return e
	} else if response.ErrorMessage != nil {
		return errors.New("Command error: " + response.ErrorCode + ": " + response.ErrorMessage.Message)
	}
	return nil
}
