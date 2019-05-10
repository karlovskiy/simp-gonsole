package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	d = flag.Bool("d", false, "Enable debug logging")
)

func main() {

	flag.Usage = func() {
		fmt.Print(`Simp TCP console client

Usage:
  simp-gonsole server_address username

Arguments:
  server_address    server address, for example: localhost:7777		
  username          your nickname

Flags:
  simp-gonsole -d ...   Enable debug logging
`)
	}

	flag.Parse()

	serverAddr := flag.Arg(0)
	if serverAddr == "" {
		flag.Usage()
		logFatal("server address must be specified\n")
	}

	username := flag.Arg(1)
	if username == "" {
		flag.Usage()
		logFatal("username must be specified\n")
	}

	logDebug("connecting to %v by %q\n", serverAddr, username)

	tcpAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		logFatal("error resolving server address: %v\n", err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logFatal("error connection to server: %v\n", err)
	}

	go readHandler(conn, username)

	user := []byte(username)
	userLen := len(user)

	connect := make([]byte, 3)
	connect[0] = 1 // version
	connect[1] = 0 // connect
	connect[2] = byte(userLen)

	connect = append(connect, user...)
	_, err = conn.Write(connect)
	if err != nil {
		logFatal("error send connect request to server: %v\n", err)
	}
	logDebug("connect request sent: %v\n", connect)

	console("CLIENT: %q connected to %v\n", username, serverAddr)
	cReader := bufio.NewReader(os.Stdin)
	for {
		text, err := cReader.ReadString('\n')
		if err != nil {
			logError("error reading string from console: %v\n", err)
			break
		}
		if text == "\n" {
			console("CLIENT: exiting...\n")
			break
		}

		connect[1] = 2 // message
		message := []byte(strings.ReplaceAll(text, "\n", ""))
		messageLen := byte(len(message))

		messageSize := make([]byte, 4)
		messageSize[0] = messageLen >> 24 & 0xFF
		messageSize[1] = messageLen >> 16 & 0xFF
		messageSize[2] = messageLen >> 8 & 0xFF
		messageSize[3] = messageLen & 0xFF
		messageData := append(connect, messageSize...)
		messageData = append(messageData, message...)

		_, err = conn.Write(messageData)
		if err != nil {
			logFatal("error send message to server: %v\n", err)
		}
		logDebug("connect request sent: %v\n", connect)
	}

	_ = conn.Close()
}

func readHandler(conn *net.TCPConn, username string) {
	r := bufio.NewReader(conn)
	for {
		respType, err := readResponseType(r)
		if err != nil {
			logError("error reading response type: %v\n", err)
			break
		}
		logDebug("received response type: %d\n", respType)

		if respType == 1 {
			ul, err := readConnectSuccessfully(r)
			if err != nil {
				logError("error reading connect successfully: %v\n", err)
				break
			}
			console("SERVER: online users: %v\n", ul)
		} else if respType == 2 || respType == 3 {
			u, err := readUser(r)
			if err != nil {
				logError("error reading user: %v\n", err)
				break
			}
			if respType == 2 {
				console("SERVER: new user connected: %q\n", u)
			} else {
				console("SERVER: user disconnected: %q\n", u)
			}
		} else if respType == 4 {
			u, err := readUser(r)
			if err != nil {
				logError("error reading user: %v\n", err)
				break
			}
			m, err := readMessage(r)
			if err != nil {
				logError("error reading message: %v\n", err)
				break
			}
			console("%s: %s\n", u, m)
		} else if respType == 0 {
			errorCode, err := readError(r)
			if err != nil {
				logError("error reading error: %v\n", err)
				break
			}
			if errorCode == 1 {
				console("SERVER: user %q is already connected\n", username)
			} else if errorCode == 0 {
				console("SERVER: server unavailable\n")
			} else {
				logError("unsupported error code: %d\n", errorCode)
			}
			break
		}
	}
	_ = conn.Close()
	os.Exit(1)
}

func readError(r io.Reader) (byte, error) {
	errorCodeBuff := make([]byte, 1)
	_, err := io.ReadFull(r, errorCodeBuff)
	if err != nil {
		return 0, err
	}
	return errorCodeBuff[0], nil
}

func readMessage(r io.Reader) (string, error) {
	mSizeBuff := make([]byte, 4)
	_, err := io.ReadFull(r, mSizeBuff)
	if err != nil {
		return "", err
	}
	mSize := (mSizeBuff[0]&0xFF)<<24 |
		(mSizeBuff[1]&0xFF)<<16 |
		(mSizeBuff[2]&0xFF)<<8 |
		(mSizeBuff[3] & 0xFF)
	mBuff := make([]byte, mSize)
	_, err = io.ReadFull(r, mBuff)
	if err != nil {
		return "", err
	}
	return string(mBuff), nil
}

func readUser(r io.Reader) (string, error) {
	uSizeBuff := make([]byte, 1)
	_, err := io.ReadFull(r, uSizeBuff)
	if err != nil {
		return "", err
	}
	uBuff := make([]byte, uSizeBuff[0])
	_, err = io.ReadFull(r, uBuff)
	if err != nil {
		return "", err
	}
	return string(uBuff), nil
}

func readConnectSuccessfully(r io.Reader) (string, error) {
	ulSizeBuff := make([]byte, 2)
	_, err := io.ReadFull(r, ulSizeBuff)
	if err != nil {
		return "", err
	}
	ulSize := (ulSizeBuff[0]&0xFF)<<8 | (ulSizeBuff[1] & 0xFF)
	ulBuff := make([]byte, ulSize)
	_, err = io.ReadFull(r, ulBuff)
	if err != nil {
		return "", err
	}
	return string(ulBuff), nil
}

func readResponseType(r io.Reader) (byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return 0, err
	}
	version := header[0]
	if version != 1 {
		return 0, errors.New(fmt.Sprintf("unsupported protocol version: %d", version))
	}
	respType := header[1]
	if respType > 4 || respType < 0 {
		return 0, errors.New(fmt.Sprintf("unsupported response type: %d", respType))
	}
	return respType, nil
}

func console(format string, v ...interface{}) {
	date := fmt.Sprintf("[%s] ", time.Now().Format("15:04:05"))
	fmt.Printf(date+format, v...)
}

func logFatal(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func logError(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func logDebug(format string, v ...interface{}) {
	if *d {
		log.Printf(format, v...)
	}
}
