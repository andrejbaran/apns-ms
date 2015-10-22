package apns

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"io"
	"net"
	"time"
)

const (
	// APNSGatewayProduction ...
	APNSGatewayProduction = "gateway.push.apple.com"

	// APNSGatewaySandbox ...
	APNSGatewaySandbox = "gateway.sandbox.push.apple.com"

	// APNSGatewayPort ...
	APNSGatewayPort uint16 = 2195

	// FeedbackGatewayProduction ...
	FeedbackGatewayProduction = "feedback.push.apple.com"

	// FeedbackGatewaySandbox ...
	FeedbackGatewaySandbox = "feedback.sandbox.push.apple.com"

	// FeedbackGatewayPort ...
	FeedbackGatewayPort uint16 = 2196
)

var (
	apnsGatewayProduction     = APNSGatewayProduction
	apnsGatewaySandbox        = APNSGatewaySandbox
	apnsGatewayPort           = APNSGatewayPort
	feedbackGatewayProduction = FeedbackGatewayProduction
	feedbackGatewaySandbox    = FeedbackGatewaySandbox
	feedbackGatewayPort       = FeedbackGatewayPort
)

func setupWorkerCommandLineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&apnsGatewayProduction, "apns-gate-production", apnsGatewayProduction, "FQDN of Apple's APNS production gateway.")
	fs.StringVar(&apnsGatewaySandbox, "apns-gate-sandbox", apnsGatewaySandbox, "FQDN of Apple's APNS sandbox gateway.")
	fs.StringVar(&feedbackGatewayProduction, "feedback-gate-production", feedbackGatewayProduction, "FQDN of Apple's Feedback service production gateway.")
	fs.StringVar(&feedbackGatewaySandbox, "feedback-gate-sandbox", feedbackGatewaySandbox, "FQDN of Apple's Feedback service sandbox gateway.")
	fs.Uint16Var(&apnsGatewayPort, "apns-gate-port", apnsGatewayPort, "Apple's APNS port number")
	fs.Uint16Var(&feedbackGatewayPort, "feedback-gate-port", feedbackGatewayPort, "Apple's Feedback service port number")
}

// worker ...
type worker struct {
	id int

	tlsConfig *tls.Config
	tlsConn   *tls.Conn

	readySignal chan bool
	pauseSignal chan bool
	quitSignal  chan bool
	errorSignal chan CommandErrorInterface

	workQueue chan CommandInterface
}

// newWorker creates, initializes and returns new worker
func newWorker(workerID int, c *Client) (w *worker, err error) {
	w = new(worker)

	w.id = workerID

	w.readySignal = make(chan bool, 1)
	w.pauseSignal = make(chan bool, 1)
	w.quitSignal = make(chan bool)
	w.errorSignal = make(chan CommandErrorInterface)

	w.workQueue = make(chan CommandInterface)

	logger.Debugf("Initializing worker #%d", workerID)
	err = w.init(c)

	return
}

func (w *worker) init(c *Client) (err error) {

	var gateway string
	if c.isProdEnv() {
		gateway = apnsGatewayProduction
	} else {
		gateway = apnsGatewaySandbox
	}

	config := &tls.Config{
		ServerName:   gateway,
		Certificates: []tls.Certificate{c.certificate},
	}

	logger.Debugf("Worker #%d TLS config %+v", w.id, config)
	w.tlsConfig = config

	err = w.connect()

	if err != nil {
		return
	}

	w.readySignal <- true

	go func() {
		for {
			select {
			case err := <-w.errorSignal:
				select {
				case c.commandErrorsQueue <- err:
					break
				default:
					logger.Errorf("Worker #%d encountered error and either nobody is listening or error queue is full: %+v", w.id, err)
				}
			}
		}
	}()

	// execute commands from queue
	logger.Debugf("Worker #%d Starting Command execution routine", w.id)
	go w.executionLoopRoutine(c)

	return
}

func (w *worker) connect() (err error) {
	var conn net.Conn

	dialer := &net.Dialer{}
	dialer.KeepAlive = time.Second * 10

	logger.Infof("Worker #%d connecting to %s:%d", w.id, w.tlsConfig.ServerName, apnsGatewayPort)

	conn, err = dialer.Dial("tcp", fmt.Sprintf("%s:%d", w.tlsConfig.ServerName, apnsGatewayPort))
	if err != nil {
		// fmt.Println("worker: error dialing ...", err)
		return
	}

	logger.Debugf("Worker #%d connected to %s", w.id, conn.RemoteAddr().String())

	w.tlsConn = tls.Client(conn, w.tlsConfig)
	err = w.tlsConn.Handshake()

	if err != nil {
		// fmt.Println("worker: error in tls ...", err)
		return
	}

	return
}

func (w *worker) disconnect() {
	logger.Warningf("Worker #%d disconnecting", w.id)
	w.tlsConn.Close()
}

func (w *worker) reconnect() {
	logger.Warningf("Worker #%d reconnecting", w.id)

	logger.Debugf("Worker #%d is pausing", w.id)
	w.pauseSignal <- true

	go func() {
		w.disconnect()
		err := w.connect()

		if err != nil {
			//TODO: Better solution!?
			commandError := NewCommandError(err, nil)
			w.errorSignal <- commandError
			w.quitSignal <- true
			return
		}

		logger.Debugf("Worker #%d continues after reconnection", w.id)
		w.readySignal <- true
	}()
}

func (w *worker) executeCommand(cmd CommandInterface) (err error) {
	var read, wrote int
	var cmdBytes []byte
	var responseBytes = make([]byte, 6)

	logger.Infof("Worker #%d processing %s", w.id, cmd)

	cmdBytes, err = cmd.Bytes()
	if err != nil {
		return
	}

	// write data to APNS
	logger.Debugf("Worker #%d writing %+v bytes", w.id, len(cmdBytes))
	// w.tlsConn.SetWriteDeadline(time.Now().Add(time.Millisecond * 1000))
	wrote, err = w.tlsConn.Write(cmdBytes)
	logger.Debugf("Worker #%d wrote %d bytes", w.id, wrote)

	if err != nil {
		logger.Debugf("Worker #%d failed to write %d bytes", w.id, len(cmdBytes))

		if err == io.EOF {
			logger.Warningf("Worker #%d connection appears to be closed by peer", w.id)
			err = errors.New("apns/worker: Error writing data. Connection appears to be closed by peer")
			w.reconnect()
		}

		return
	}

	// read response from APNS
	w.tlsConn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
	read, err = w.tlsConn.Read(responseBytes)
	logger.Debugf("Worker #%d read %d bytes %+v", w.id, read, responseBytes)

	if err != nil {
		logger.Debugf("Worker #%d read error: %s", w.id, err)

		if err == io.EOF {
			logger.Warningf("Worker #%d connection closed by peer", w.id)
		}

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			err = nil
		}
	}

	if read > 0 {
		logger.Warningf("Worker #%d received error response", w.id)

		commandError := NewCommandErrorFromAPNSResponse(responseBytes, cmd)
		w.errorSignal <- commandError

		select {
		case cmd.Errors() <- commandError:
			break
		default:
			break
		}
	}

	if read > 0 || err == io.EOF {
		w.reconnect()

		if err == io.EOF {
			err = errors.New("apns/worker: Connection was closed by peer after reading data")
		}
	}

	return
}

func (w *worker) executionLoopRoutine(c *Client) {
	defer w.disconnect()

	for {
		select {
		case <-w.readySignal:
			logger.Debugf("Worker #%d ready", w.id)

			c.workerQueue <- w.workQueue
			logger.Debugf("Worker #%d added itself to worker queue", w.id)
			logger.Infof("Worker #%d waiting for commands", w.id)

			select {
			case command := <-w.workQueue:
				startTime := time.Now()
				err := w.executeCommand(command)
				endTime := time.Now()

				logger.Infof("Worker #%d processed %s in %s", w.id, command, endTime.Sub(startTime))

				if err != nil {
					commandError := NewCommandError(err, command)
					w.errorSignal <- commandError

					select {
					case command.Errors() <- commandError:
						break
					default:
						break
					}

				}

				select {
				case <-w.pauseSignal:
					logger.Warningf("Worker #%d received pause signal", w.id)
					break
				default:
					w.readySignal <- true
				}

				close(command.Errors())
			}

			break

		case <-w.pauseSignal:
			logger.Warningf("Worker #%d pausing execution loop", w.id)
			break

		case <-w.quitSignal:
			// TODO: Restart worker!
			// defer w.restart()
			return
		}
	}
}
