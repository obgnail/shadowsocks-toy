package server

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	"github.com/obgnail/shadowsocks-toy/connection"
	"github.com/obgnail/shadowsocks-toy/logger"
	log "github.com/sirupsen/logrus"
	"net"
)

type Server struct {
	cipher     cipher.Cipher
	listenAddr *net.TCPAddr
}

func New(listenAddr string, c cipher.Cipher) (*Server, error) {
	if c == nil {
		c = &cipher.NopCipher{}
	}

	lAddr, err := net.ResolveTCPAddr("tcp4", listenAddr)
	if err != nil {
		return nil, err
	}
	return &Server{listenAddr: lAddr, cipher: c}, nil
}

func (s *Server) Listen(didListen func(listenAddr *net.TCPAddr)) error {
	listener, err := net.ListenTCP("tcp", s.listenAddr)
	if err != nil {
		return errors.Trace(err)
	}
	log.Info("Server Listen ", fmt.Sprintf(logger.GreenBackWhiteTextFormat, listener.Addr()))
	if didListen != nil {
		go didListen(s.listenAddr)
	}
	s.startListen(listener)
	return nil
}

func (s *Server) startListen(listener net.Listener) {
	for {
		userConn, err := listener.Accept()
		if err != nil {
			continue
		}
		log.Debugf("%s -> %s | %s -> %s", logger.ClientStr, logger.ServerStr, userConn.RemoteAddr(), userConn.LocalAddr())
		go s.handleConn(userConn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	userConn := connection.NewSecureSocket(conn, s.cipher)
	defer userConn.Close()

	dstConn, err := s.sock5(userConn)
	if err != nil {
		log.Error(errors.Trace(err))
		return
	}
	defer dstConn.Close()

	log.Debugf(
		"%s <-> %s <-> %s | %s <-> %s(%s) <-> %s",
		logger.ClientStr, logger.ServerStr, logger.TargetStr,
		userConn.RemoteAddr(), userConn.LocalAddr(), dstConn.LocalAddr(), dstConn.RemoteAddr(),
	)
	err = connection.Tunnel(userConn, dstConn)
	if err != nil {
		log.Error(errors.Trace(err))
	}
}

func (s *Server) sock5(userConn *connection.SecureSocket) (dstConn *connection.SecureSocket, err error) {
	_, err = connection.HandShakeHandler(userConn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	dst, _, err := connection.RequestHandler(userConn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	log.Debugf("%s -> %s | %s -> %s", logger.ServerStr, logger.TargetStr, dst.LocalAddr(), dst.RemoteAddr())
	dstConn = connection.NewSecureSocket(dst, s.cipher)
	return dstConn, nil
}
