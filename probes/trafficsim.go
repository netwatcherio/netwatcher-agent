package probes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

// file paths
const (
	privateKeyPath                                   = "private_key.pem"
	certificatePath                                  = "certificate.pem"
	keySize                                          = 2048 // Recommended size for RSA keys
	TrafficSimMsgType_Registration TrafficSimPayload = "registration"
	TrafficSimMsgType_Payload      TrafficSimPayload = "payload"
	SimMsgSize                                       = 128 // does this need to be higher? what will our marshaled json be in size? keeping at 1024 to be safe
)

type TrafficSimPayload string

type TrafficSimMsg struct {
	Type    TrafficSimPayload  `json:"type"`    // type of message, eg registration, etc
	Agent   primitive.ObjectID `json:"agent"`   // if sending to a server, it will be the agent id of client, if sending to a client, it will be the agent id of the server
	From    primitive.ObjectID `json:"from"`    // if replying to a message from a client, it will be the same but in reverse
	Payload string             `json:"payload"` // the actual data
}

type TrafficSimType string

const (
	TrafficSimType_Client TrafficSimType = "client"
	TrafficSimType_Server TrafficSimType = "server"
)

type TrafficSim struct {
	Running     bool
	Errored     bool
	DataSend    chan string
	DataReceive chan string
	Conn        *quic.Connection
	Stream      *quic.Stream
	ThisAgent   primitive.ObjectID
	OtherAgent  primitive.ObjectID
	IPAddress   string
	Port        string // make this int?
	Type        TrafficSimType
	Registered  bool
}

/*func TrafficSimClient(pp *Probe) error {
	// targetHost := strings.Split(pp.Config.Target[0].Target, ":")

	err := checkAndGenerateCertificateIfNeeded()
	if err != nil {
		return err
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"netwatcher-agent"},
	}
	conn, err := quic.DialAddr(context.Background(), pp.Config.Target[0].Target, tlsConf, nil)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	fmt.Printf("Client: Sending '%s'\n", message)
	_, err = stream.Write([]byte(message))
	if err != nil {
		return err
	}

	buf := make([]byte, len(message))
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return err
	}
	fmt.Printf("Client: Got '%s'\n", buf)

	return nil
}*/

/*func TrafficSimServer(pp *Probe) error {
	// targetHost := strings.Split(pp.Config.Target[0].Target, ":")

	// todo handle errors better?
	go func() {

		listener, err := quic.ListenAddr(pp.Config.Target[0].Target, generateTLSConfig(), nil)
		if err != nil {
			log.Errorf(err.Error())
		}
		defer listener.Close()

		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Errorf(err.Error())
		}

		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			panic(err)
		}
		defer stream.Close()

		_, err = io.Copy(loggingWriter{stream}, stream)
	}()
	return nil
}*/

// file paths and key size remain unchanged

func (sim *TrafficSim) sendMessage(msg *TrafficSimMsg) error {
	bytes, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	_, err = (*sim.Stream).Write(bytes)
	return err
}

// below is the certificate bull shiet

func checkAndGenerateCertificateIfNeeded() error {
	if !fileExists(privateKeyPath) || !fileExists(certificatePath) {
		if err := generatePrivateKey(privateKeyPath, keySize); err != nil {
			return err
		}
		if err := generateCertificate(certificatePath, privateKeyPath); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func generatePrivateKey(filePath string, keySize int) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Failed to close file: %v", err)
		}
	}(file)

	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	return pem.Encode(file, privatePEM)
}

func generateCertificate(certPath, privateKeyPath string) error {
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"netwatcher.io agent"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer func(certFile *os.File) {
		err := certFile.Close()
		if err != nil {
			log.Errorf("Failed to close file: %v", err)
		}
	}(certFile)

	return pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyPEMBlock, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyPEMBlock)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, err
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func generateTLSConfig() *tls.Config {
	if err := checkAndGenerateCertificateIfNeeded(); err != nil {
		log.Fatalf("Failed to check/generate certificate: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
	if err != nil {
		log.Fatalf("Failed to load x509 key pair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"netwatcher-agent"},
	}
}
