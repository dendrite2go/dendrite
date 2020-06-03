package trusted

import (
	log "log"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
)

// Redeclare event types, so that they can be extended with event handler methods.
type TrustedKeyAddedEvent struct {
	grpc_config.TrustedKeyAddedEvent
}
type TrustedKeyRemovedEvent struct {
	grpc_config.TrustedKeyRemovedEvent
}
type KeyManagerAddedEvent struct {
	grpc_config.KeyManagerAddedEvent
}
type KeyManagerRemovedEvent struct {
	grpc_config.KeyManagerRemovedEvent
}

// Projection

type Projection struct {
	TrustedKeys    map[string]string
	KeyManagers    map[string]string
	AggregateState axon_utils.AggregateState
}

func (projection *Projection) GetAggregateState() axon_utils.AggregateState {
	return projection.AggregateState
}

func RestoreProjection(aggregateIdentifier string, clientConnection *axon_utils.ClientConnection) *Projection {
	return axon_utils.RestoreProjection("Trusted Keys", aggregateIdentifier, createInitialProjection, clientConnection, prepareUnmarshal).(*Projection)
}

func createInitialProjection() interface{} {
	return &Projection{
		TrustedKeys:    make(map[string]string),
		KeyManagers:    make(map[string]string),
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
	log.Printf("Configuration Projection: Payload type: %v", payloadType)
	switch payloadType {
	case "TrustedKeyAddedEvent":
		event = &TrustedKeyAddedEvent{}
	case "TrustedKeyRemovedEvent":
		event = &TrustedKeyRemovedEvent{}
	case "KeyManagerAddedEvent":
		event = &KeyManagerAddedEvent{}
	case "KeyManagerRemovedEvent":
		event = &KeyManagerRemovedEvent{}
	default:
		event = nil
	}
	return event
}

// Event Handlers

func (event *TrustedKeyAddedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.TrustedKeys[event.PublicKey.Name] = event.PublicKey.PublicKey
}

func (event *TrustedKeyRemovedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.TrustedKeys[event.Name] = ""
}

func (event *KeyManagerAddedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.KeyManagers[event.PublicKey.Name] = event.PublicKey.PublicKey
}

func (event *KeyManagerRemovedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.KeyManagers[event.Name] = ""
}
