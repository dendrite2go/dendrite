package trusted

import (
	log "log"

	proto "github.com/golang/protobuf/proto"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
)

const AggregateIdentifier = "trusted-keys-aggregate"

func HandleRegisterTrustedKeyCommand(commandMessage *axon_server.Command, clientConnection *axon_utils.ClientConnection) (*axon_utils.Error, error) {
	command := grpc_config.RegisterTrustedKeyCommand{}
	e := proto.Unmarshal(commandMessage.Payload.Data, &command)
	if e != nil {
		log.Printf("Could not unmarshal RegisterTrustedKeyCommand")
		return nil, e
	}

	projection := RestoreProjection(AggregateIdentifier, clientConnection)

	currentValue := projection.TrustedKeys[command.PublicKey.Name]
	newValue := command.PublicKey.PublicKey
	if newValue == currentValue {
		return nil, nil
	}

	var eventType string
	var event axon_utils.Event
	if len(newValue) > 0 {
		eventType = "TrustedKeyAddedEvent"
		event = &TrustedKeyAddedEvent{
			grpc_config.TrustedKeyAddedEvent{
				PublicKey: command.PublicKey,
			},
		}
	} else {
		eventType = "TrustedKeyRemovedEvent"
		event = &TrustedKeyRemovedEvent{
			grpc_config.TrustedKeyRemovedEvent{
				Name: command.PublicKey.Name,
			},
		}
	}
	log.Printf("Trusted aggregate: emit: %v: %v", eventType, event)
	return axon_utils.AppendEvent(event, AggregateIdentifier, projection, clientConnection)
}

func HandleRegisterKeyManagerCommand(commandMessage *axon_server.Command, clientConnection *axon_utils.ClientConnection) (*axon_utils.Error, error) {
	command := grpc_config.RegisterKeyManagerCommand{}
	e := proto.Unmarshal(commandMessage.Payload.Data, &command)
	if e != nil {
		log.Printf("Could not unmarshal RegisterKeyManagerCommand")
		return nil, e
	}

	projection := RestoreProjection(AggregateIdentifier, clientConnection)

	currentValue := projection.KeyManagers[command.PublicKey.Name]
	newValue := command.PublicKey.PublicKey
	if newValue == currentValue {
		return nil, nil
	}

	var eventType string
	var event axon_utils.Event
	if len(newValue) > 0 {
		eventType = "KeyManagerAddedEvent"
		event = &KeyManagerAddedEvent{
			grpc_config.KeyManagerAddedEvent{
				PublicKey: command.PublicKey,
			},
		}
	} else {
		eventType = "KeyManagerRemovedEvent"
		event = &KeyManagerRemovedEvent{
			grpc_config.KeyManagerRemovedEvent{
				Name: command.PublicKey.Name,
			},
		}
	}
	log.Printf("Trusted aggregate: emit: %v: %v", eventType, event)
	return axon_utils.AppendEvent(event, AggregateIdentifier, projection, clientConnection)
}
