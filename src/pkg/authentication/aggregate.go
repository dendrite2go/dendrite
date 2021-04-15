package authentication

import (
	log "log"

	proto "github.com/golang/protobuf/proto"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/dendrite_config"
	trusted "github.com/dendrite2go/dendrite/src/pkg/trusted"
)

const AggregateIdentifier = "credentials-aggregate"

func HandleRegisterUnencryptedCredentialsCommand(commandMessage *axon_server.Command, clientConnection *axon_utils.ClientConnection) (*axon_utils.Error, error) {
	command := grpc_config.RegisterUnencryptedCredentialsCommand{}
	e := proto.Unmarshal(commandMessage.Payload.Data, &command)
	if e != nil {
		log.Printf("Could not unmarshal RegisterUnencryptedCredentialsCommand")
		return nil, e
	}
	credentials := command.GetCredentials()
	credentials.Secret, e = trusted.EncryptString(credentials.Secret)
	if e != nil {
		log.Printf("Could not encrypt secret")
		return nil, e
	}
	return handleRegisterEncryptedCredentialsCommand(credentials, clientConnection)
}

func HandleRegisterCredentialsCommand(commandMessage *axon_server.Command, clientConnection *axon_utils.ClientConnection) (*axon_utils.Error, error) {
	command := grpc_config.RegisterCredentialsCommand{}
	e := proto.Unmarshal(commandMessage.Payload.Data, &command)
	if e != nil {
		log.Printf("Could not unmarshal RegisterCredentialsCommand")
		return nil, e
	}
	credentials := command.GetCredentials()
	return handleRegisterEncryptedCredentialsCommand(credentials, clientConnection)
}

func handleRegisterEncryptedCredentialsCommand(credentials *grpc_config.Credentials, clientConnection *axon_utils.ClientConnection) (*axon_utils.Error, error) {
	projection := RestoreProjection(AggregateIdentifier, clientConnection)

	if CheckKnown(credentials, projection) {
		return nil, nil
	}

	var eventType string
	var event axon_utils.Event
	if len(credentials.Secret) > 0 {
		eventType = "CredentialsAddedEvent"
		event = &CredentialsAddedEvent{
			grpc_config.CredentialsAddedEvent{
				Credentials: credentials,
			},
		}
	} else {
		eventType = "CredentialsRemovedEvent"
		event = &CredentialsRemovedEvent{
			grpc_config.CredentialsRemovedEvent{
				Identifier: credentials.Identifier,
			},
		}
	}
	log.Printf("Credentials aggregate: emit: %v: %v", eventType, event)
	return axon_utils.AppendEvent(event, AggregateIdentifier, projection, clientConnection)
}
