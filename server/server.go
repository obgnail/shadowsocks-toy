package server

import (
	"encoding/binary"
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
	log.Info(fmt.Sprintf(logger.GreenBackWhiteTextFormat, "Server Listen"), " ", listener.Addr())
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
		log.Infof("%s -> %s | %s -> %s", logger.ClientStr, logger.ServerStr, userConn.RemoteAddr(), userConn.LocalAddr())
		go s.handleConn(userConn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	userConn := connection.NewSecureSocket(conn, s.cipher)
	defer userConn.Close()

	dstConn, err := s.sock5(userConn)
	if err != nil {
		log.Error(errors.Trace(err))
	}
	defer dstConn.Close()

	err = connection.Tunnel(userConn, dstConn)
	if err != nil {
		log.Error(errors.Trace(err))
	}
}

func (s *Server) sock5(userConn *connection.SecureSocket) (dstConn *connection.SecureSocket, err error) {
	buf := make([]byte, 256)

	/**
	   The localConn connects to the dstServer, and sends a ver
	   identifier/method selection message:
		          +----+----------+----------+
		          |VER | NMETHODS | METHODS  |
		          +----+----------+----------+
		          | 1  |    1     | 1 to 255 |
		          +----+----------+----------+
	   The VER field is set to X'05' for this ver of the protocol.  The
	   NMETHODS field contains the number of method identifier octets that
	   appear in the METHODS field.
	*/
	if _, err = userConn.DecryptToBytes(buf); err != nil {
		return nil, errors.Trace(err)
	}

	// 第一个字段VER代表Socks的版本，Socks5默认为0x05，其固定长度为1个字节
	// 只支持版本5
	if VER := buf[0]; VER != 0x05 {
		return nil, fmt.Errorf("support sock5 only")
	}

	/**
	   The dstServer selects from one of the methods given in METHODS, and
	   sends a METHOD selection message:
		          +----+--------+
		          |VER | METHOD |
		          +----+--------+
		          | 1  |   1    |
		          +----+--------+
	*/
	// 不需要验证，直接验证通过
	resp := []byte{0x05, 0x00}
	if _, err := userConn.EncryptFromBytes(resp); err != nil {
		return nil, errors.Trace(err)
	}

	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/

	// 获取真正的远程服务的地址
	buf = make([]byte, 256)
	n, err := userConn.DecryptToBytes(buf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// n 最短的长度为7 情况为 ATYP=3 DST.ADDR占用1字节 值为0x0
	if n < 7 {
		return nil, fmt.Errorf("error n: %d", n)
	}

	// CMD代表客户端请求的类型，值长度也是1个字节，有三种类型
	// CONNECT X'01',目前只支持 CONNECT
	if CMD := buf[1]; CMD != 0x01 {
		return nil, fmt.Errorf("error CMD: %d", buf[1])
	}

	var dIP []byte
	// aType 代表请求的远程服务器地址类型，值长度1个字节，有三种类型
	switch ATYP := buf[3]; ATYP {
	case 0x01:
		//	IP V4 address: X'01'
		dIP = buf[4 : 4+net.IPv4len]
	case 0x03:
		//	DOMAINNAME: X'03'
		ipAddr, err := net.ResolveIPAddr("ip", string(buf[5:n-2]))
		if err != nil {
			return nil, errors.Trace(err)
		}
		dIP = ipAddr.IP
	case 0x04:
		//	IP V6 address: X'04'
		dIP = buf[4 : 4+net.IPv6len]
	default:
		return nil, fmt.Errorf("no such ATYP: %d", buf[3])
	}

	dPort := buf[n-2 : n]
	dstAddr := &net.TCPAddr{
		IP:   dIP,
		Port: int(binary.BigEndian.Uint16(dPort)),
	}

	dst, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Conn被关闭时直接清除所有数据 不管没有发送的数据
	dst.SetLinger(0)

	log.Infof("%s -> %s | %s -> %s", logger.ServerStr, logger.TargetStr, dst.LocalAddr(), dst.RemoteAddr())

	// 响应客户端连接成功
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	// 响应客户端连接成功
	if _, err := userConn.EncryptFromBytes([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return nil, errors.Trace(err)
	}
	dstConn = connection.NewSecureSocket(dst, s.cipher)
	return dstConn, nil
}
