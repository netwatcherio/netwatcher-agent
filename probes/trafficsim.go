package probes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

// file paths
const (
	privateKeyPath  = "private_key.pem"
	publicKeyPath   = "public_key.pem"
	certificatePath = "certificate.pem"
	keySize         = 2048 // Recommended size for RSA keys
)

func TrafficSimServer(pp *Probe) error {
	// targetHost := strings.Split(pp.Config.Target[0].Target, ":")

	listener, err := quic.ListenAddr(pp.Config.Target[0].Target, generateTLSConfig(), nil)
	if err != nil {
		return err
	}
	defer listener.Close()

	conn, err := listener.Accept(context.Background())
	if err != nil {
		return err
	}

	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	_, err = io.Copy(loggingWriter{stream}, stream)
	return err
}

func InitTrafficSimServer() {
	err := checkAndGenerateCertificateIfNeeded()
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}
}

// A wrapper for io.Writer that also logs the message.
type loggingWriter struct{ io.Writer }

func (w loggingWriter) Write(b []byte) (int, error) {
	fmt.Printf("Server: Got '%s'\n", string(b))
	return w.Writer.Write(b)
}

// file paths and key size remain unchanged

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
	defer file.Close()

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
	defer certFile.Close()

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
