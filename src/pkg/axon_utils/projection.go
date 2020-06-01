package axon_utils

import (
	context "context"
	errors "errors"
	io "io"
	log "log"

	proto "github.com/golang/protobuf/proto"

	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
)

type Event interface {
	proto.Message
	ApplyTo(projection interface{})
}

type CachedProjection interface {
	GetAggregateState() AggregateState
}

type AggregateState interface {
	GetSequenceNumber() int64
	SetSequenceNumber(int64)
}

type aggregateState struct {
	sequenceNumber int64
}

type Cache interface {
	Get(string) (interface{}, bool)
	Put(string, interface{})
	Delete(string)
}

var cache Cache = nil

func SetCache(theCache Cache) error {
	if cache != nil {
		return errors.New("cache already set")
	}
	cache = theCache
	return nil
}

func (s *aggregateState) GetSequenceNumber() int64 {
	return s.sequenceNumber
}

func (s *aggregateState) SetSequenceNumber(n int64) {
	s.sequenceNumber = n
}

func NewAggregateState() AggregateState {
	return &aggregateState{sequenceNumber: -1}
}

func RestoreProjection(label string, aggregateIdentifier string, createInitialProjection func() interface{}, clientConnection *ClientConnection, prepareUnmarshal func(payloadType string) Event) interface{} {
	var projection interface{}
	if cache != nil {
		projection, found := cache.Get(aggregateIdentifier)
		if found {
			log.Printf("Restore projection: cache hit: %v", aggregateIdentifier)
			return projection
		}
	}

	projection = createInitialProjection()
	log.Printf("%v Projection: %v", label, projection)

	eventStoreClient := axon_server.NewEventStoreClient(clientConnection.Connection)
	requestEvents := axon_server.GetAggregateEventsRequest{
		AggregateId:     aggregateIdentifier,
		InitialSequence: -1,
		AllowSnapshots:  false,
	}
	log.Printf("%v Projection: Request events: %v", label, requestEvents)
	client, e := eventStoreClient.ListAggregateEvents(context.Background(), &requestEvents)
	if e != nil {
		log.Printf("%v Projection: Error while requesting aggregate events: %v", label, e)
		return nil
	}
	var lastSequenceNumber int64
	lastSequenceNumber = -1
	for {
		eventMessage, e := client.Recv()
		if e == io.EOF {
			log.Printf("%v Projection: End of stream", label)
			break
		} else if e != nil {
			log.Printf("%v Projection: Error while receiving next event: %v", label, e)
			break
		}
		log.Printf("%v Projection: Received event: %v", label, eventMessage)
		lastSequenceNumber = eventMessage.AggregateSequenceNumber
		if eventMessage.Payload != nil {
			payloadType := eventMessage.Payload.Type
			event := prepareUnmarshal(payloadType)
			if event == nil {
				log.Printf("%v Projection: unrecognized payload type: %v", label, payloadType)
				continue
			}
			e := proto.Unmarshal(eventMessage.Payload.Data, event.(proto.Message))
			if e != nil {
				log.Printf("%v Projection: Could not unmarshal %v", label, eventMessage.Payload.Type)
			}
			event.ApplyTo(projection)
		}
	}
	if cache != nil {
		switch p := projection.(type) {
		case CachedProjection:
			log.Printf("%v Projection: Put in cache: %v: %v", label, aggregateIdentifier, lastSequenceNumber)
			p.GetAggregateState().SetSequenceNumber(lastSequenceNumber)
		}
		cache.Put(aggregateIdentifier, projection)
	}
	return projection
}

func cacheEvict(aggregateIdentifier string) {
	if cache != nil {
		cache.Delete(aggregateIdentifier)
	}
}
