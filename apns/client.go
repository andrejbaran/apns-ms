package apns

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"io"
	"net"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	// CommandsQueueSize specifies default notifications queue size
	CommandsQueueSize = 100000
)

var (
	env                              = "sandbox"
	commandsQueueSize         uint64 = CommandsQueueSize
	numberOfWorkers                  = uint32(runtime.NumCPU() * 2)
	certifcateFile            string
	certificatePrivateKeyFile string
	workerID                  uint32
)

func setupClientCommandLineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&env, "env", env, "Environment of Apple's APNS and Feedback service gateways. For production use specify \"production\", for testing specify \"sandbox\".")
	fs.Uint64Var(&commandsQueueSize, "max-notifications", commandsQueueSize, "Number of notification that can be queued for processing at once. Once the queue is full all requests to raw push notification endpoint will result in 503 Service Unavailable response.")
	fs.Uint32Var(&numberOfWorkers, "workers", numberOfWorkers, "Number of workers that concurently process push notifications. Defaults to 2 * Number of CPU cores.")
	fs.StringVar(&certifcateFile, "cert", certifcateFile, "Absolute path to certificate file. Certificate is expected be in PEM format.")
	fs.StringVar(&certificatePrivateKeyFile, "cert-key", certificatePrivateKeyFile, "Absolute path to certificate private key file. Certificate key is expected be in PEM format.")
}

// ClientConfig holds some configuration options for Client
type ClientConfig struct {
	// Env is either "production" or "sandbox"
	Env string

	// NumberOfWorkers sets number of workers for sending push notifications
	NumberOfWorkers uint32

	// CertificateFile is absolute path to APNS certificate file
	CertificateFile string

	// CertificatePrivateKey is absolute path to APNS certificate private key file
	CertificatePrivateKeyFile string

	// CommandsQueueSize sets the queue size for push notifications
	CommandsQueueSize uint64
}

// NewClientConfig returns new client config
func NewClientConfig() (config *ClientConfig) {
	config = new(ClientConfig)
	config.Env = env
	config.NumberOfWorkers = numberOfWorkers
	config.CommandsQueueSize = commandsQueueSize
	config.CertificateFile = certifcateFile
	config.CertificatePrivateKeyFile = certificatePrivateKeyFile

	return
}

// Client struct is the main class for interacting with Apple Push Notification Service
type Client struct {
	Config             *ClientConfig
	certificate        tls.Certificate
	commandsQueue      chan CommandInterface
	workerQueue        chan chan CommandInterface
	commandErrorsQueue chan CommandErrorInterface
}

// NewClient creates a new Client
func NewClient(config *ClientConfig) (client *Client, err error) {
	client = nil
	err = nil

	logger.Debugf("Setting up client")
	logger.Debugf("Client config: %+v", config)

	// validate and create certificate
	logger.Debug("Validating certificate files...")
	var certificate tls.Certificate
	certificate, err = tls.LoadX509KeyPair(config.CertificateFile, config.CertificatePrivateKeyFile)

	if err != nil {
		logger.Fatalf("Error was encountered during certificate validation: %s", err)
		return
	}

	// setup channels
	logger.Debugf("Setting up command queue: %+v", config.CommandsQueueSize)
	nCh := make(chan CommandInterface, config.CommandsQueueSize)

	logger.Debugf("Setting up workers queue: %+v", config.NumberOfWorkers)
	wCh := make(chan chan CommandInterface, config.NumberOfWorkers)

	logger.Debugf("Setting up command errors queue: %+v", config.CommandsQueueSize)
	eCh := make(chan CommandErrorInterface, config.CommandsQueueSize)
	err = nil

	// client
	client = new(Client)

	client.Config = config
	client.certificate = certificate
	client.commandsQueue = nCh
	client.workerQueue = wCh
	client.commandErrorsQueue = eCh

	err = client.init()
	if err != nil {
		logger.Fatal(err)
	}

	return
}

// Errors returns a channel to consume command errors
func (c *Client) Errors() <-chan CommandErrorInterface {
	return c.commandErrorsQueue
}

func (c *Client) init() (err error) {
	var i uint32
	err = nil

	logger.Infof("Initializing %d worker(s)", c.Config.NumberOfWorkers)

	for i = 0; i < c.Config.NumberOfWorkers; i++ {
		atomic.AddUint32(&workerID, 1)
		worker, workerErr := newWorker(int(workerID), c)
		if workerErr != nil {
			//TODO issue warning about this and try to create the worker again later
			logger.Warningf("Worker #%d couldn't be initialized: %s", worker.id, workerErr)
		} else {
			// logger.Infof("%s%+v %s", "Worker #", worker.id, "ready")
		}
	}

	logger.Debugf("Starting client dispatcher routines")

	// errors
	go func() {
		for {
			select {
			case commandError := <-c.commandErrorsQueue:
				go func() {
					//TODO logging
					logger.Warningf("Received error: %s for command %s", commandError, commandError.GetCommand())
				}()
			}
		}
	}()

	// main dispatch loop
	go func() {
		for {
			select {
			case cmd := <-c.commandsQueue:
				go func() {
					logger.Debugf("Received command from queue %+v", cmd)
					select {
					case workerWorkQueue := <-c.workerQueue:
						logger.Debugf("Forwarding command to worker")
						workerWorkQueue <- cmd
						break

					}
				}()
			}
		}
	}()

	return
}

// ExecuteCommand queues command for execution
func (c *Client) ExecuteCommand(cmd CommandInterface) error {
	select {
	case c.commandsQueue <- cmd:
		logger.Debugf("Scheduled %s for execution", cmd)
		break

	default:
		close(cmd.Errors())
		logger.Warningf("Command queue is full, dropping command: %s", cmd)
		return NewCommandError(errors.New("apns: Queue is full, dismissing command"), cmd)
	}

	return nil
}

// CheckFeedbackService connects to Apple's feedback service and returns FeedbackResponse object
func (c *Client) CheckFeedbackService() (rsp *FeedbackResponse, err error) {
	var conn net.Conn
	var read int
	var responseBytes = make([]byte, 38)

	dialer := &net.Dialer{}
	dialer.KeepAlive = time.Second * 10

	var gateway string
	if c.isProdEnv() {
		gateway = FeedbackGatewayProduction
	} else {
		gateway = FeedbackGatewaySandbox
	}

	tlsConfig := &tls.Config{}
	tlsConfig.ServerName = gateway
	tlsConfig.Certificates = []tls.Certificate{c.certificate}

	logger.Infof("Connecting to %s:%d", tlsConfig.ServerName, FeedbackGatewayPort)

	conn, err = dialer.Dial("tcp", fmt.Sprintf("%s:%d", tlsConfig.ServerName, FeedbackGatewayPort))
	if err != nil {
		logger.Error("Error connecting feedback service")
		return
	}

	logger.Debugf("Connected to %s", conn.RemoteAddr().String())

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		logger.Error("Error establishing tls connection to feedback service")
		return
	}

	rsp = new(FeedbackResponse)

	for {
		tlsConn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		read, err = tlsConn.Read(responseBytes)
		logger.Debugf("Read %d bytes %+v", read, responseBytes)

		if read > 0 {
			rsp.addEntryFromBytes(responseBytes)
		}

		if err != nil {
			if err == io.EOF {
				logger.Info("Read all data from feedback service and connection was closed by peer")
				err = nil
				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				err = nil
				return
			}

			logger.Warningf("Error reading response from feedback service: %s", err)
		}
	}

	return
}

func (c *Client) isProdEnv() bool {
	return c.Config.Env == "production"
}
