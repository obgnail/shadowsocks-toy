package client

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	"github.com/obgnail/shadowsocks-toy/connection"
	"github.com/obgnail/shadowsocks-toy/logger"
	log "github.com/sirupsen/logrus"
	"net"
)

type Client struct {
	cipher     cipher.Cipher
	localAddr  *net.TCPAddr
	serverAddr *net.TCPAddr
}

func New(listenAddr, remoteAddr string, c cipher.Cipher) (*Client, error) {
	if c == nil {
		c = &cipher.NopCipher{}
	}

	lAddr, err := net.ResolveTCPAddr("tcp4", listenAddr)
	if err != nil {
		return nil, err
	}
	rAddr, err := net.ResolveTCPAddr("tcp4", remoteAddr)
	if err != nil {
		return nil, err
	}
	return &Client{localAddr: lAddr, serverAddr: rAddr, cipher: c}, nil
}

func (c *Client) Listen(didListen func(listenAddr *net.TCPAddr)) error {
	local, err := net.ListenTCP("tcp", c.localAddr)
	if err != nil {
		return errors.Trace(err)
	}

	log.Info(fmt.Sprintf(logger.GreenBackWhiteTextFormat, "Client Listen"), " ", local.Addr())
	if didListen != nil {
		go didListen(c.localAddr)
	}
	c.startListen(local)
	return nil
}

func (c *Client) startListen(listener net.Listener) {
	for {
		localConn, err := listener.Accept()
		if err != nil {
			continue
		}
		log.Infof("%s  -> %s | %s -> %s", logger.LocalStr, logger.ClientStr, localConn.RemoteAddr(), localConn.LocalAddr())
		go c.handleConn(localConn)
	}
}

func (c *Client) handleConn(conn net.Conn) {
	localConn := connection.NewSecureSocket(conn, c.cipher)
	defer localConn.Close()

	server, err := net.Dial("tcp", c.serverAddr.String())
	if err != nil {
		return
	}

	serverConn := connection.NewSecureSocket(server, c.cipher)
	defer serverConn.Close()

	err = connection.Tunnel(serverConn, localConn)
	if err != nil {
		log.Error(errors.ErrorStack(err))
	}
}
