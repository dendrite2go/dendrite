package main

import (
	bufio "bufio"
	context "context"
	fmt "fmt"
	log "log"
	os "os"
	strings "strings"

	rand "crypto/rand"
	rsa "crypto/rsa"

	ssh "golang.org/x/crypto/ssh"
	grpc "google.golang.org/grpc"

	grpc_example "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
	trusted "github.com/dendrite2go/dendrite/src/pkg/trusted"
	utils "github.com/dendrite2go/dendrite/src/pkg/utils"
)

func main() {
	log.Printf("Key Manager -- reading from standard input")

	conn, e := grpc.Dial("host.docker.internal:8181", grpc.WithInsecure())
	if e != nil {
		log.Printf("Error when dialing application: %v", e)
		panic("Could not connect to application")
	}
	defer utils.ReportError("Close connection to application", conn.Close)

	client := grpc_example.NewConfigurationServiceClient(conn)
	log.Printf("Greeter service client: %v", client)

	stream, e := client.ChangeTrustedKeys(context.Background())
	if e != nil {
		log.Printf("Error when opening stream for ChangeTrustedKeys: %v", e)
		panic("Could not call ChangeTrustedKeys")
	}
	log.Printf("ChangeTrustedKeys stream: %v", stream)

	request := grpc_example.TrustedKeyRequest{}
	e = stream.Send(&request)
	if e != nil {
		log.Printf("Error when sending first (empty) request for ChangeTrustedKeys: %v", e)
		panic("Could not send first request")
	}

	response, e := stream.Recv()
	if e != nil {
		log.Printf("Error when receiving first response for ChangeTrustedKeys: %v", e)
		panic("Could not receive first response]")
	}
	log.Printf("First response: %v", response)
	nonce := response.Nonce

	reader := bufio.NewReader(os.Stdin)
	line := readLine(reader)
	var name string
	var pem string
	var managerPrivateKey *rsa.PrivateKey
	var signer ssh.Signer
	var signatureName string
	for true {
		if strings.HasPrefix(line, ">>> Manager: ") {
			signatureName = getName(line)
			pem, line = readPem(reader)
			log.Printf("Manager name: %v: %d", signatureName, len(pem))
			managerPrivateKey, e = trusted.ParsePrivateKey(pem)
			if e != nil {
				log.Printf("Error when parsing manager private key: %v", e)
				panic("Could not parse manager private key")
			}
			signer, e = ssh.NewSignerFromKey(managerPrivateKey)
			if e != nil {
				log.Printf("Unable to create signer from private key: %v", e)
				panic("Could not create signer")
			}
		} else if strings.HasPrefix(line, ">>> Management") {
			line, nonce = addPublicKeys(true, reader, &signer, signatureName, nonce, stream)
		} else if strings.HasPrefix(line, ">>> Trusted") {
			line, nonce = addPublicKeys(false, reader, &signer, signatureName, nonce, stream)
		} else if strings.HasPrefix(line, ">>> Identity Provider: ") {
			name = getName(line)
			pem, line = readPem(reader)
			log.Printf("Identity Provider name: %v: %d", name, len(pem))
			grpcPrivateKey := grpc_example.PrivateKey{
				Name:       name,
				PrivateKey: pem,
			}
			_, e = client.SetPrivateKey(context.Background(), &grpcPrivateKey)
			if e != nil {
				log.Printf("Error when setting private key for identity provider: %v", e)
				panic("Could not set private key")
			}
		} else if strings.HasPrefix(line, ">>> Secrets") {
			line = addSecrets(client, reader, signatureName, &signer)
		} else if strings.HasPrefix(line, ">>> Properties") {
			line = addProperties(client, reader)
		} else if strings.HasPrefix(line, ">>> End") {
			log.Printf("End")
			request.PublicKey = nil
			request.Nonce = nonce
			request.Signature = nil
			request.IsKeyManager = false
			e = stream.Send(&request)
			if e != nil {
				log.Printf("Error when sending final request: %v", e)
				panic("Could not send final request")
			}
			e = stream.CloseSend()
			if e != nil {
				log.Printf("Error when receiving closing send directions: %v", e)
				panic("Could not close send direction")
			}
			response, e = stream.Recv()
			if e != nil {
				log.Printf("Error when receiving final response: %v", e)
				panic("Could not receive final response")
			}
			log.Printf("Final response: %v: %v: %v", response.Status.Code, response.Status.Message, response.Nonce)
			break
		} else {
			panic("Expected: >>> { Manager: <name> | Management | Trusted | Identity Provider: <name> | End }: got: [" + line + "]")
		}
	}
}

func readLine(reader *bufio.Reader) string {
	text, e := reader.ReadString('\n')
	if e != nil {
		m, _ := fmt.Printf("Error reading string from stdin: %v", e)
		panic(m)
	}
	return text
}

func getName(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return "???"
	} else {
		return strings.Trim(parts[1], " \t\r\n")
	}
}

func readPem(reader *bufio.Reader) (pem string, nextLine string) {
	pem = ""
	var builder strings.Builder
	for true {
		nextLine = readLine(reader)
		if strings.HasPrefix(nextLine, ">>> ") {
			break
		}
		builder.WriteString(nextLine)
	}
	pem = builder.String()
	return
}

func addPublicKeys(isKeyManager bool, reader *bufio.Reader, signer *ssh.Signer, signatureName string, startNonce []byte, stream grpc_example.ConfigurationService_ChangeTrustedKeysClient) (line string, nonce []byte) {
	nonce = startNonce
	for true {
		line = readLine(reader)
		if strings.HasPrefix(line, ">>> ") {
			return
		}
		parts := strings.Split(strings.Trim(line, "\n"), " ")
		if len(parts) < 3 {
			log.Printf("Not enough parts: %v", line)
			continue
		}
		log.Printf("Trusted public key: %v: %v", parts[2], parts[1])
		publicKey := grpc_example.PublicKey{
			Name:      parts[2],
			PublicKey: parts[1],
		}
		log.Printf("Public key: %v", publicKey)
		signature, e := (*signer).Sign(rand.Reader, nonce)
		if e != nil {
			log.Printf("Unable to sign nonce: %v", e)
			panic("Could not sign nonce")
		}
		log.Printf("Signature: %v", signature)
		grpcSignature := grpc_example.Signature{
			Format:        signature.Format,
			Blob:          signature.Blob,
			Rest:          signature.Rest,
			SignatureName: signatureName,
		}
		request := grpc_example.TrustedKeyRequest{
			PublicKey:    &publicKey,
			Nonce:        nonce,
			Signature:    &grpcSignature,
			IsKeyManager: isKeyManager,
		}
		e = stream.Send(&request)
		if e != nil {
			log.Printf("Error when sending add key request for ChangeTrustedKeys: %v", e)
			panic("Could not send first request")
		}

		response, e := stream.Recv()
		if e != nil {
			log.Printf("Error when receiving response for ChangeTrustedKeys: %v", e)
			panic("Could not receive first response")
		}
		log.Printf("Response: %v", response)
		nonce = response.Nonce
	}
	return "", nonce
}

func addSecrets(client grpc_example.ConfigurationServiceClient, reader *bufio.Reader, signatureName string, signer *ssh.Signer) (line string) {
	log.Printf("Add secrets: client: %v", client)
	stream, e := client.ChangeCredentials(context.Background())
	if e != nil {
		log.Printf("Add secrets: Error when opening stream: %v", e)
	}
	log.Printf("Add secrets: stream: %v", stream)
	defer utils.ReportError("Close connection to application", func() error { _, e := stream.CloseAndRecv(); return e })

	for true {
		line = readLine(reader)
		if strings.HasPrefix(line, ">>>") {
			break
		}
		log.Printf("Add secret: %v", line)
		parts := strings.SplitN(line, "=", 2)
		log.Printf("Number of parts: %v", len(parts))

		signature, e := (*signer).Sign(rand.Reader, []byte(line))
		if e != nil {
			log.Printf("Add secrets: Unable to sign nonce: %v", e)
			panic("Could not sign nonce")
		}
		log.Printf("Add secrets: Signature: %v", signature)
		grpcSignature := grpc_example.Signature{
			Format:        signature.Format,
			Blob:          signature.Blob,
			Rest:          signature.Rest,
			SignatureName: signatureName,
		}

		credentials := grpc_example.Credentials{
			Identifier: parts[0],
			Secret:     parts[1],
			Signature:  &grpcSignature,
		}
		log.Printf("Add secret: Credentials: %v", credentials)
		e = stream.Send(&credentials)
		if e != nil {
			log.Printf("Add secrets: Unable to send credentials: %v", e)
			panic("Could not send credentials")
		}
	}
	emptyCredentials := grpc_example.Credentials{
		Identifier: "",
		Secret:     "",
		Signature:  nil,
	}
	e = stream.Send(&emptyCredentials)
	if e != nil {
		log.Printf("Error while sending empty credentials: %v", e)
	}
	return
}

func addProperties(client grpc_example.ConfigurationServiceClient, reader *bufio.Reader) (line string) {
	log.Printf("Add properties: client: %v", client)

	for true {
		line = readLine(reader)
		if strings.HasPrefix(line, ">>>") {
			break
		}
		log.Printf("Add property: %v", line)
		parts := strings.SplitN(line, "=", 2)
		log.Printf("Number of parts: %v", len(parts))
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		keyValue := grpc_example.KeyValue{
			Key:   key,
			Value: value,
		}
		log.Printf("Add property: KeyValue: %v", keyValue)
		_, e := client.SetProperty(context.Background(), &keyValue)
		if e != nil {
			log.Printf("Add properties: Unable to set property: %v: %v", key, e)
			panic("Could not set property")
		}
	}
	return
}
