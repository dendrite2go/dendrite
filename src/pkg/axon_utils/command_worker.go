package axon_utils

import (
	log "log"

	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
)

type Error struct {
	Code                string
	Message             string
	AggregateIdentifier string
}

func CommandWorker(stream axon_server.CommandService_OpenStreamClient, clientConnection *ClientConnection, commandDispatcher func(*axon_server.Command, axon_server.CommandService_OpenStreamClient, *ClientConnection) (*Error, error)) {
	clientId := clientConnection.ClientInfo.ClientId
	for true {
		CommandAddPermits(1, stream, clientId)

		log.Printf("Command worker: Waiting for command")
		inbound, e := stream.Recv()
		if e != nil {
			log.Printf("Command worker: Error on receive: %v", e)
			break
		}
		log.Printf("Command worker: Inbound: %v", inbound)
		command := inbound.GetCommand()
		if command != nil {
			commandName := command.Name
			log.Printf("Command worker: Received %v", commandName)
			e = handleCommand(stream, clientConnection, command, commandDispatcher)
			if e != nil {
				return
			}
		}
	}
}

func handleCommand(stream axon_server.CommandService_OpenStreamClient, clientConnection *ClientConnection, command *axon_server.Command, commandDispatcher func(*axon_server.Command, axon_server.CommandService_OpenStreamClient, *ClientConnection) (*Error, error)) error {
	var commandError *Error
	var e error

	defer endCommand(stream, &commandError, command.MessageIdentifier)

	for i := 0; i < 10; i++ {
		log.Printf("Handle command: Dispatch command: %v: %v", command.Name, command.MessageIdentifier)
		commandError, e = commandDispatcher(command, stream, clientConnection)
		if e != nil {
			log.Printf("Command worker: Error on dispatch: %v: %v", command.Name, e)
			commandError = &Error{
				Code:                "FAULT",
				Message:             "Internal error",
				AggregateIdentifier: "CommandMessage:" + command.MessageIdentifier,
			}
			return e
		}
		if commandError == nil || commandError.Code != "" {
			log.Printf("Handle command: Finished: %v", commandError)
			break
		}
		log.Printf("Handle command: Evict from cache: %v", commandError.AggregateIdentifier)
		cacheEvict(commandError.AggregateIdentifier)
	}
	return nil
}

func endCommand(stream axon_server.CommandService_OpenStreamClient, commandError **Error, messageIdentifier string) {
	log.Printf("End command: %v: %v", messageIdentifier, *commandError)
	if *commandError == nil {
		CommandRespond(stream, messageIdentifier)
	} else if (*commandError).Code == "" {
		ReportError(stream, messageIdentifier, "FAILED", "Command failed 10 times")
	} else {
		ReportError(stream, messageIdentifier, (*commandError).Code, (*commandError).Message)
	}
}
