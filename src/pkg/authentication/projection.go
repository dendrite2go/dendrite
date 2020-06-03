package authentication

import (
	log "log"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
)

// Redeclare event types, so that they can be extended with event handler methods.
type CredentialsAddedEvent struct {
	grpc_config.CredentialsAddedEvent
}
type CredentialsRemovedEvent struct {
	grpc_config.CredentialsRemovedEvent
}

// Projection

type Projection struct {
	Credentials    map[string]string
	AggregateState axon_utils.AggregateState
}

func RestoreProjection(aggregateIdentifier string, clientConnection *axon_utils.ClientConnection) *Projection {
	return axon_utils.RestoreProjection("Authentication", aggregateIdentifier, createInitialProjection, clientConnection, prepareUnmarshal).(*Projection)
}

func createInitialProjection() interface{} {
	return &Projection{
		Credentials:    make(map[string]string),
		AggregateState: axon_utils.NewAggregateState(),
	}
}

func (projection *Projection) Apply(event axon_utils.Event) {
	event.ApplyTo(projection)
}

// Map an event name as stored in AxonServer to an object that has two aspects:
// 1. It is a proto.Message, so it can be unmarshalled from the Axon event
// 2. It is an axon_utils.Event, so it can be applied to a projection
func prepareUnmarshal(payloadType string) (event axon_utils.Event) {
	log.Printf("Credentials Projection: Payload type: %v", payloadType)
	switch payloadType {
	case "CredentialsAddedEvent":
		event = &CredentialsAddedEvent{}
	case "CredentialsRemovedEvent":
		event = &CredentialsRemovedEvent{}
	default:
		event = nil
	}
	return event
}

// Event Handlers

func (event *CredentialsAddedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.Credentials[event.Credentials.Identifier] = event.Credentials.Secret
}

func (event *CredentialsRemovedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.Credentials[event.Identifier] = ""
}
