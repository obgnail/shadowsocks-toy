package client

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	"github.com/obgnail/shadowsocks-toy/connection"
	"github.com/obgnail/shadowsocks-toy/logger"
	"github.com/obgnail/shadowsocks-toy/ruleset"
	log "github.com/sirupsen/logrus"
	"net"
)

type Client struct {
	ruleset    ruleset.Ruleset
	cipher     cipher.Cipher
	localAddr  *net.TCPAddr
	serverAddr *net.TCPAddr
}

func New(listenAddr, remoteAddr string, c cipher.Cipher, r ruleset.Ruleset) (*Client, error) {
	if c == nil {
		c = &cipher.NopCipher{}
	}
	if r == nil {
		r = &ruleset.Global{}
	}

	lAddr, err := net.ResolveTCPAddr("tcp4", listenAddr)
	if err != nil {
		return nil, err
	}
	rAddr, err := net.ResolveTCPAddr("tcp4", remoteAddr)
	if err != nil {
		return nil, err
	}
	return &Client{localAddr: lAddr, serverAddr: rAddr, cipher: c, ruleset: r}, nil
}

func (c *Client) Listen(didListen func(listenAddr *net.TCPAddr)) error {
	local, err := net.ListenTCP("tcp", c.localAddr)
	if err != nil {
		return errors.Trace(err)
	}

	log.Info("Client Listen ", fmt.Sprintf(logger.GreenBackWhiteTextFormat, local.Addr()))
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
		log.Debugf("%s -> %s | %s -> %s", logger.LocalStr, logger.ClientStr, localConn.RemoteAddr(), localConn.LocalAddr())
		go func() {
			err := c.handleConn(localConn)
			log.Error(errors.ErrorStack(err))
		}()
	}
}

func (c *Client) handleConn(conn net.Conn) (err error) {
	defer conn.Close()
	local := connection.NewSecureSocket(conn, cipher.NewNopCipher())
	handshakeReceived, requestReceived, dst, err := connection.GetSock5Data(local)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("%s -> %s | %s -> %s", logger.ClientStr, logger.ServerStr, dst.LocalAddr(), dst.RemoteAddr())
	// dont use server to proxy conn
	targetIP := dst.RemoteAddr()
	if !c.ruleset.Match(targetIP.(*net.TCPAddr)) {
		return errors.Trace(c.joinTarget(conn, targetIP.String()))
	}
	// use server to proxy conn
	return errors.Trace(c.joinServer(conn, handshakeReceived, requestReceived))
}

func (c *Client) joinTarget(localConn net.Conn, dstAddr string) error {
	targetConn, err := net.Dial("tcp", dstAddr)
	if err != nil {
		return errors.Trace(err)
	}
	defer targetConn.Close()
	log.Debugf(
		"%s <-> %s <-> %s | %s <-> %s(%s) <-> %s",
		logger.LocalStr, logger.ClientStr, logger.TargetStr,
		localConn.RemoteAddr(), localConn.LocalAddr(), targetConn.LocalAddr(), targetConn.RemoteAddr(),
	)
	return errors.Trace(connection.Copy(localConn, targetConn))
}

func (c *Client) joinServer(conn net.Conn, handshakeReceived, requestReceived []byte) error {
	localConn := connection.NewSecureSocket(conn, c.cipher)
	server, err := net.Dial("tcp", c.serverAddr.String())
	if err != nil {
		return errors.Trace(err)
	}
	serverConn := connection.NewSecureSocket(server, c.cipher)
	defer serverConn.Close()
	if err := connection.SendSocks5Data(serverConn, handshakeReceived, requestReceived); err != nil {
		return errors.Trace(err)
	}
	log.Debugf(
		"%s <-> %s <-> %s | %s <-> %s(%s) <-> %s",
		logger.LocalStr, logger.ClientStr, logger.ServerStr,
		localConn.RemoteAddr(), localConn.LocalAddr(), serverConn.LocalAddr(), serverConn.RemoteAddr(),
	)
	return errors.Trace(connection.Tunnel(serverConn, localConn))
}
