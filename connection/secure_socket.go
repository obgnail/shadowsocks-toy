package connection

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
)

const (
	UseClosedConnErr   = "use of closed network connection"
	ConnResetByPeerErr = "connection reset by peer"
)

var recoverableNetError = errors.New("recoverable net error")

type SecureSocket struct {
	net.Conn
	cipher cipher.Cipher

	closeFlag bool
}

func NewSecureSocket(conn net.Conn, cipher cipher.Cipher) *SecureSocket {
	ss := &SecureSocket{
		Conn:      conn,
		cipher:    cipher,
		closeFlag: false,
	}
	return ss
}

func (ss *SecureSocket) Hijack() net.Conn {
	return ss.Conn
}

func (ss *SecureSocket) Close() (err error) {
	if ss == nil || ss.closeFlag {
		return
	}
	if err = ss.Conn.Close(); err == nil {
		ss.closeFlag = true
	}
	return
}

func (ss *SecureSocket) DecryptTo(to io.Writer) error {
	return Decrypt(ss, to)
}

func (ss *SecureSocket) EncryptTo(to io.Writer) error {
	return Encrypt(ss, to)
}

func (ss *SecureSocket) DecryptToBytes(to []byte) (int, error) {
	return DecryptToBytes(ss, to)
}

func (ss *SecureSocket) EncryptFromBytes(from []byte) (int, error) {
	return EncryptFromBytes(from, ss)
}

type cipherFunc func(from *SecureSocket, to io.Writer) error

// from --(encrypt)--> to
func Encrypt(from *SecureSocket, to io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	for {
		readCount, err := from.Conn.Read(buf)
		if err != nil {
			return handlerNetError(err)
		}
		if readCount > 0 {
			data, err := from.cipher.Encrypt(buf[:readCount])
			if err != nil {
				return errors.Trace(err)
			}
			if _, err := to.Write(data); err != nil {
				return handlerNetError(err)
			}
		}
	}
}

// from --(decrypt)--> to
func Decrypt(from *SecureSocket, to io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	for {
		readCount, err := from.Conn.Read(buf)
		if err != nil {
			return handlerNetError(err)
		}
		if readCount > 0 {
			data, err := from.cipher.Decrypt(buf[:readCount])
			if err != nil {
				return errors.Trace(err)
			}
			if _, err := to.Write(data); err != nil {
				return handlerNetError(err)
			}
		}
	}
}

// from --(decrypt)--> to
func DecryptToBytes(from *SecureSocket, to []byte) (int, error) {
	temp := make([]byte, len(to))
	n, err := from.Conn.Read(temp)
	if err != nil {
		return n, handlerNetError(err)
	}
	temp, err = from.cipher.Decrypt(temp[:n])
	if err != nil {
		err = errors.Trace(err)
	}
	copy(to, temp)
	return len(temp), err
}

// from --(encrypt)--> to
func EncryptFromBytes(from []byte, to *SecureSocket) (int, error) {
	encryptData, err := to.cipher.Encrypt(from)
	if err != nil {
		return 0, errors.Trace(err)
	}
	_, err = to.Conn.Write(encryptData)
	if err != nil {
		return 0, handlerNetError(err)
	}
	return len(from), nil
}

// join plainConn and cipherConn, block until error occurs:
// plain ---(encrypt)--> cipher
// plain <--(decrypt)--- cipher
func Tunnel(cipher, plain *SecureSocket) error {
	errChan := make(chan error, 2)
	pipe := func(cipherFunc cipherFunc, from, to *SecureSocket) {
		defer to.Close()
		defer from.Close()
		if err := cipherFunc(from, to); err != nil {
			if err == recoverableNetError {
				//log.Debug(errors.Trace(err))
			} else {
				errChan <- errors.Trace(err)
			}
		}
	}

	go pipe(Decrypt, cipher, plain)
	go pipe(Encrypt, plain, cipher)
	// ignore second error
	return <-errChan
}

// join two conn, block until error occurs
func Copy(c1, c2 net.Conn) error {
	errChan := make(chan error, 2)
	pipe := func(c1, c2 net.Conn) {
		defer c1.Close()
		defer c2.Close()
		_, err := io.Copy(c1, c2)
		err = ignoreNetError(err)
		if err != nil {
			errChan <- errors.Trace(err)
		}
	}

	go pipe(c1, c2)
	go pipe(c2, c1)
	return <-errChan
}

func handlerNetError(err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return recoverableNetError
	}
	if e, ok := err.(*net.OpError); ok && strings.Contains(e.Err.Error(), UseClosedConnErr) {
		return recoverableNetError
	}
	if strings.Contains(err.Error(), ConnResetByPeerErr) {
		log.Warn(err)
		return nil
	}
	return errors.Trace(err)
}

func ignoreNetError(err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil
	}
	if e, ok := err.(*net.OpError); ok && strings.Contains(e.Err.Error(), UseClosedConnErr) {
		return nil
	}
	if strings.Contains(err.Error(), ConnResetByPeerErr) {
		log.Warn(err)
		return nil
	}
	return errors.Trace(err)
}

// 测试用
func PrintContent(conn net.Conn) {
	buf := make([]byte, 256)
	count, err := conn.Read(buf)
	if err != nil {
		log.Error(errors.Trace(err))
	}
	fmt.Println("count:\t", count)
	fmt.Println("buf:\t", buf)
	fmt.Println("string(buf):\t", string(buf))
	fmt.Println()
}
