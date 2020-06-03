package example_api

import (
	context "context"
	errors "errors"
	log "log"
	time "time"

	rand "crypto/rand"
	hex "encoding/hex"

	jwt "github.com/pascaldekloe/jwt"
	grpc "google.golang.org/grpc"

	authentication "github.com/dendrite2go/dendrite/src/pkg/authentication"
	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
	trusted "github.com/dendrite2go/dendrite/src/pkg/trusted"
	utils "github.com/dendrite2go/dendrite/src/pkg/utils"
)

type GreeterServer struct {
	conn       *grpc.ClientConn
	clientInfo *axon_server.ClientIdentification
}

var empty = grpc_config.Empty{}

func (s *GreeterServer) Authorize(_ context.Context, credentials *grpc_config.Credentials) (*grpc_config.AccessToken, error) {
	accessToken := grpc_config.AccessToken{
		Jwt: "",
	}
	if authentication.Authenticate(credentials.Identifier, credentials.Secret) {
		var claims jwt.Claims
		claims.Subject = credentials.Identifier
		claims.Issued = jwt.NewNumericTime(time.Now().Round(time.Second))
		token, e := trusted.CreateJWT(claims)
		if e != nil {
			return nil, e
		}
		accessToken.Jwt = token
	}
	return &accessToken, nil
}

func (s *GreeterServer) ListTrustedKeys(_ *grpc_config.Empty, streamServer grpc_config.ConfigurationService_ListTrustedKeysServer) error {
	trustedKey := grpc_config.PublicKey{}
	for name, key := range trusted.GetTrustedKeys() {
		trustedKey.Name = name
		trustedKey.PublicKey = key
		log.Printf("Server: Trusted keys streamed reply: %v", trustedKey)
		_ = streamServer.Send(&trustedKey)
		log.Printf("Server: Trusted keys streamed reply sent")
	}
	return nil
}

func (s *GreeterServer) SetPrivateKey(_ context.Context, request *grpc_config.PrivateKey) (*grpc_config.Empty, error) {
	_ = trusted.SetPrivateKey(request.Name, request.PrivateKey)

	empty := grpc_config.Empty{}
	return &empty, nil
}

func (s *GreeterServer) ChangeTrustedKeys(stream grpc_config.ConfigurationService_ChangeTrustedKeysServer) error {
	var status = grpc_config.Status{}
	response := grpc_config.TrustedKeyResponse{}
	nonce := make([]byte, 64)
	first := true
	for true {
		request, e := stream.Recv()
		if e != nil {
			log.Printf("Server: Change trusted keys: error receiving request: %v", e)
			return e
		}

		status.Code = 500
		status.Message = "Internal Server Error"

		if first {
			first = false
			status.Code = 200
			status.Message = "OK"
		} else {
			if request.Signature == nil {
				status.Code = 200
				status.Message = "End of stream"
				response.Status = &status
				response.Nonce = nil
				_ = stream.Send(&response)
				return nil
			}
			configRequest := grpc_config.TrustedKeyRequest{}
			if e := utils.ProtoCast(request, &configRequest); e != nil {
				return e
			}
			e = trusted.AddTrustedKey(&configRequest, nonce, toClientConnection(s))
			if e == nil {
				status.Code = 200
				status.Message = "OK"
			} else {
				status.Code = 400
				status.Message = e.Error()
			}
		}

		_, _ = rand.Reader.Read(nonce)
		hexNonce := hex.EncodeToString(nonce)
		log.Printf("Next nonce: %v", hexNonce)

		response.Status = &status
		response.Nonce = nonce
		e = stream.Send(&response)
		if e != nil {
			log.Printf("Server: Change trusted keys: error sending response: %v", e)
			return e
		}
	}
	return errors.New("server: Change trusted keys: unexpected end of stream")
}

func (s *GreeterServer) ChangeCredentials(stream grpc_config.ConfigurationService_ChangeCredentialsServer) error {
	for true {
		credentials, e := stream.Recv()
		if e != nil {
			log.Printf("Error while receiving credentials: %v", e)
			return e
		}
		if credentials.Signature == nil {
			break
		}
		configCredentials := grpc_config.Credentials{}
		if e := utils.ProtoCast(credentials, &configCredentials); e != nil {
			return e
		}
		_ = authentication.SetCredentials(&configCredentials, toClientConnection(s))
	}
	empty = grpc_config.Empty{}
	return stream.SendAndClose(&empty)
}

func (s *GreeterServer) SetProperty(_ context.Context, keyValue *grpc_config.KeyValue) (*grpc_config.Empty, error) {
	log.Printf("Server: Set property: %v: %v", keyValue.Key, keyValue.Value)

	command := grpc_config.ChangePropertyCommand{
		Property: keyValue,
	}
	e := axon_utils.SendCommand("ChangePropertyCommand", &command, toClientConnection(s))
	if e != nil {
		log.Printf("Trusted: Error when sending ChangePropertyCommand: %v", e)
	}

	empty = grpc_config.Empty{}
	return &empty, nil
}

func RegisterWithServer(grpcServer *grpc.Server, clientConnection *axon_utils.ClientConnection) {
	grpc_config.RegisterConfigurationServiceServer(grpcServer, &GreeterServer{clientConnection.Connection, clientConnection.ClientInfo})
}

func toClientConnection(s *GreeterServer) *axon_utils.ClientConnection {
	result := axon_utils.ClientConnection{
		Connection: s.conn,
		ClientInfo: s.clientInfo,
	}
	return &result
}
