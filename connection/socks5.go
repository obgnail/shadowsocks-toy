package connection

import (
	"encoding/binary"
	"fmt"
	"github.com/juju/errors"
	"net"
)

func HandShakeHandler(conn *SecureSocket) (received []byte, err error) {
	received = make([]byte, 256)
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
	n, err := conn.DecryptToBytes(received)
	if err != nil {
		return received, errors.Trace(err)
	}
	received = received[:n]

	// 第一个字段VER代表Socks的版本，Socks5默认为0x05，其固定长度为1个字节
	// 只支持版本5
	if VER := received[0]; VER != 0x05 {
		return received, fmt.Errorf("support sock5 only")
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
	if _, err := conn.EncryptFromBytes(resp); err != nil {
		return received, errors.Trace(err)
	}
	return received, nil
}

func RequestHandler(conn *SecureSocket) (dst *net.TCPConn, received []byte, err error) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	received = make([]byte, 256)
	n, err := conn.DecryptToBytes(received)
	if err != nil {
		err = errors.Trace(err)
		return
	}
	// n 最短的长度为7 情况为 ATYP=3 DST.ADDR占用1字节 值为0x0
	if n < 7 {
		err = fmt.Errorf("error n: %d", n)
		return
	}
	received = received[:n]

	// CMD代表客户端请求的类型，值长度也是1个字节，有三种类型
	// CONNECT X'01',目前只支持 CONNECT
	if CMD := received[1]; CMD != 0x01 {
		err = fmt.Errorf("error CMD: %d", received[1])
		return
	}

	var dIP []byte
	// aType 代表请求的远程服务器地址类型，值长度1个字节，有三种类型
	switch ATYP := received[3]; ATYP {
	case 0x01:
		//	IP V4 address: X'01'
		dIP = received[4 : 4+net.IPv4len]
	case 0x03:
		//	DOMAINNAME: X'03'
		ipAddr, err := net.ResolveIPAddr("ip", string(received[5:n-2]))
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		dIP = ipAddr.IP
	case 0x04:
		//	IP V6 address: X'04'
		dIP = received[4 : 4+net.IPv6len]
	default:
		err = fmt.Errorf("no such ATYP: %d", received[3])
		return
	}

	dPort := received[n-2 : n]
	dstAddr := &net.TCPAddr{
		IP:   dIP,
		Port: int(binary.BigEndian.Uint16(dPort)),
	}

	dst, err = net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	// Conn被关闭时直接清除所有数据 不管没有发送的数据
	dst.SetLinger(0)

	// 响应客户端连接成功
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	// 响应客户端连接成功
	if _, err := conn.EncryptFromBytes([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return nil, nil, errors.Trace(err)
	}
	return
}

func GetSock5Data(from *SecureSocket) (
	handshakeReceived, requestReceived []byte, dst *net.TCPConn, err error,
) {
	if handshakeReceived, err = HandShakeHandler(from); err != nil {
		err = errors.Trace(err)
		return
	}
	if dst, requestReceived, err = RequestHandler(from); err != nil {
		err = errors.Trace(err)
		return
	}
	return
}

func SendSocks5Data(serverConn *SecureSocket, handshakeReceived, requestReceived []byte) error {
	if _, err := serverConn.EncryptFromBytes(handshakeReceived); err != nil {
		return errors.Trace(err)
	}
	buf := make([]byte, 256)
	if _, err := serverConn.DecryptToBytes(buf); err != nil {
		return errors.Trace(err)
	}
	if buf[0] != 0x05 || buf[1] != 0x00 {
		return fmt.Errorf("error handshake resp: %b %b", buf[0], buf[1])
	}
	if _, err := serverConn.EncryptFromBytes(requestReceived); err != nil {
		return errors.Trace(err)
	}
	if _, err := serverConn.DecryptToBytes(buf); err != nil {
		return errors.Trace(err)
	}
	if string(buf[:10]) != string([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) {
		return fmt.Errorf("error request resp: %b", buf)
	}
	return nil
}
