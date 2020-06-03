package trusted

import (
	errors "errors"
	log "log"

	rand "crypto/rand"
	rsa "crypto/rsa"
	x509 "crypto/x509"
	base64 "encoding/base64"
	hex "encoding/hex"
	pem "encoding/pem"
	bigmath "math/big"

	jwt "github.com/pascaldekloe/jwt"
	ssh "golang.org/x/crypto/ssh"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/configuration"
)

var keyManagers map[string]string
var trustedKeys map[string]string
var privateKey rsa.PrivateKey
var privateKeyName string

func SetPrivateKey(name string, pemString string) error {
	var e error

	encodedPublicKey := trustedKeys[name]
	log.Printf("Trusted: Set private key: public key: %v: %v", name, encodedPublicKey)

	publicKey, e := getTrustedKey(name)
	if e != nil {
		log.Printf("Trusted: Set private key: Unable to parse public key: %v", e)
		return errors.New("invalid public key")
	}

	givenPrivateKey, e := ParsePrivateKey(pemString)
	if e != nil {
		return e
	}

	signer, e := ssh.NewSignerFromKey(givenPrivateKey)
	if e != nil {
		log.Printf("Trusted: Set private key: Unable to create signer from private key: %v", e)
		return errors.New("invalid private key")
	}
	nonce := make([]byte, 64)
	_, _ = rand.Reader.Read(nonce)
	hexNonce := hex.EncodeToString(nonce)
	log.Printf("Nonce: %v", hexNonce)
	signature, e := signer.Sign(rand.Reader, nonce)
	if e != nil {
		log.Printf("Trusted: Set private key: Unable to sign nonce: %v", e)
		return errors.New("invalid private key")
	}

	e = publicKey.Verify(nonce, signature)
	if e != nil {
		log.Printf("Trusted: Set private key: Unable to verify signature of nonce: %v", e)
		return errors.New("invalid private key")
	}

	privateKey = *givenPrivateKey
	privateKeyName = name
	log.Printf("Trusted: Set private key: private key: %v", name)
	return nil
}

func AddTrustedKey(request *grpc_config.TrustedKeyRequest, nonce []byte, clientConnection *axon_utils.ClientConnection) error {
	name := request.PublicKey.Name
	publicKey := request.PublicKey.PublicKey
	protoSignature := request.Signature
	signatureName := protoSignature.SignatureName
	isKeyManager := request.IsKeyManager

	_, e := parsePublicKey(publicKey)
	if e != nil {
		log.Printf("Trusted: Add trusted key: Unable to parse new trusted key: %v", e)
		return errors.New("invalid trusted key")
	}

	signatureKey, e := GetKeyManagerKey(signatureName)
	if e != nil {
		log.Printf("Trusted: Add trusted key: Unable to parse signature key: %v", e)
		return errors.New("invalid trusted key")
	}

	signature := ssh.Signature{
		Format: protoSignature.Format,
		Blob:   protoSignature.Blob,
		Rest:   protoSignature.Rest,
	}

	e = signatureKey.Verify(nonce, &signature)
	if e != nil {
		log.Printf("Trusted: Add trusted key: Unable to verify signature of nonce: %v", e)
		return errors.New("invalid trusted key")
	}

	trustedKeys[name] = publicKey
	if isKeyManager {
		command := grpc_config.RegisterKeyManagerCommand{
			PublicKey: request.PublicKey,
		}
		e = axon_utils.SendCommand("RegisterKeyManagerCommand", &command, clientConnection)
		if e != nil {
			log.Printf("Trusted: Error when sending RegisterKeyManagerCommand: %v", e)
		}
	} else {
		command := grpc_config.RegisterTrustedKeyCommand{
			PublicKey: request.PublicKey,
		}
		e = axon_utils.SendCommand("RegisterTrustedKeyCommand", &command, clientConnection)
		if e != nil {
			log.Printf("Trusted: Error when sending RegisterTrustedKeyCommand: %v", e)
		}
	}
	log.Printf("Added public key: %v: %v", name, publicKey)
	return nil
}

func UnsafeSetTrustedKey(publicKey *grpc_config.PublicKey) {
	if trustedKeys == nil {
		trustedKeys = make(map[string]string)
	}
	trustedKeys[publicKey.Name] = publicKey.PublicKey
}

func UnsafeSetKeyManager(publicKey *grpc_config.PublicKey) {
	if keyManagers == nil {
		keyManagers = make(map[string]string)
	}
	keyManagers[publicKey.Name] = publicKey.PublicKey
}

func GetKeyManagerKey(name string) (ssh.PublicKey, error) {
	var e error

	encodedPublicKey := keyManagers[name]
	log.Printf("Trusted: Get key manager key: %v: %v", name, encodedPublicKey)
	publicKey, e := parsePublicKey(encodedPublicKey)
	return publicKey, e
}

func EncryptString(plainText string) (string, error) {
	rsaPublicKey, e := GetRsaPublicKey()
	if e != nil {
		return "", e
	}
	log.Printf("Trusted: Encrypt string: RSA public key: %v", rsaPublicKey)
	encryptedBytes, e := rsa.EncryptPKCS1v15(rand.Reader, rsaPublicKey, []byte(plainText))
	if e != nil {
		return "", e
	}
	return base64.RawStdEncoding.EncodeToString(encryptedBytes), nil
}

func DecryptString(cryptText string) (string, error) {
	encryptedBytes, e := base64.RawStdEncoding.DecodeString(cryptText)
	if e != nil {
		return "", e
	}
	decryptedBytes, e := rsa.DecryptPKCS1v15(rand.Reader, &privateKey, encryptedBytes)
	if e != nil {
		return "", e
	}
	return string(decryptedBytes), nil
}

func getTrustedKey(name string) (ssh.PublicKey, error) {
	var e error

	encodedPublicKey := trustedKeys[name]
	log.Printf("Trusted: Get trusted key: %v: %v", name, encodedPublicKey)
	publicKey, e := parsePublicKey(encodedPublicKey)
	return publicKey, e
}

func parsePublicKey(encodedPublicKey string) (ssh.PublicKey, error) {
	publicKey, _, _, _, e := ssh.ParseAuthorizedKey([]byte("ssh-rsa " + encodedPublicKey))
	if e != nil {
		log.Printf("Trusted: Unable to parse public key: %v", e)
		return nil, e
	}
	return publicKey, nil
}

func ParsePrivateKey(pemString string) (givenPrivateKey *rsa.PrivateKey, e error) {
	privatePem, _ := pem.Decode([]byte(pemString))
	if privatePem.Type != "PRIVATE KEY" && privatePem.Type != "RSA PRIVATE KEY" {
		log.Printf("RSA private key is of the wrong type: %v", privatePem.Type)
		return nil, errors.New("wrong PEM type")
	}

	privatePemBytes := privatePem.Bytes
	if privatePemBytes == nil {
		log.Printf("Trusted: Set private key: empty PEM")
		return nil, errors.New("empty PEM")
	}

	var parsedKey interface{}
	if parsedKey, e = x509.ParsePKCS1PrivateKey(privatePemBytes); e != nil {
		if parsedKey, e = x509.ParsePKCS8PrivateKey(privatePemBytes); e != nil { // note this returns type `interface{}`
			log.Printf("Trusted: Set private key: Unable to parse RSA private key: %v", e)
			return nil, errors.New("invalid private key")
		}
	}

	givenPrivateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		log.Printf("Type assertion for RSA private key failed")
		return nil, errors.New("invalid private key")
	}
	return
}

func CreateJWT(claims jwt.Claims) (token string, e error) {
	tokenBuffer, e := claims.RSASign("RS256", &privateKey)
	token = string(tokenBuffer)
	return
}

func GetRsaPublicKey() (*rsa.PublicKey, error) {
	publicKey, e := parsePublicKey(trustedKeys[privateKeyName])
	if e != nil {
		return nil, e
	}
	buffer := publicKey.Marshal()
	w := getInt(buffer)
	keyType := string(buffer[4 : w+4])
	if keyType != "ssh-rsa" {
		return nil, errors.New("Not an ssh-rsa key: " + keyType)
	}
	buffer = buffer[w+4:]

	w = getInt(buffer)
	exponent := int(getBigInt(buffer[4 : w+4]).Int64())
	buffer = buffer[w+4:]

	w = getInt(buffer)
	modulus := getBigInt(buffer[4 : w+4])

	rsaPublicKey := rsa.PublicKey{
		N: modulus,
		E: exponent,
	}

	return &rsaPublicKey, e
}

func getInt(buffer []byte) (result uint32) {
	result = 0
	for i := 0; i < 4; i++ {
		result = result*256 + uint32(buffer[i])
	}
	return
}

func getBigInt(buffer []byte) *bigmath.Int {
	result := bigmath.NewInt(0)
	for _, v := range buffer {
		result.Mul(result, bigmath.NewInt(256))
		result.Add(result, bigmath.NewInt(int64(uint32(v))))
	}
	return result
}

func GetTrustedKeys() map[string]string {
	var result = make(map[string]string)
	for name, key := range trustedKeys {
		result[name] = key
	}
	return result
}
