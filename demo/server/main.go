package main

import (
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	_ "github.com/obgnail/shadowsocks-toy/logger"
	"github.com/obgnail/shadowsocks-toy/server"
	log "github.com/sirupsen/logrus"
)

const (
	ClientListenAddr = "127.0.0.1:4444"
	ServerListenAddr = "127.0.0.1:5555"
)

func main() {
	cph := cipher.NewBase64Cipher()
	//cph := cipher.NewNopCipher()

	srv, err := server.New(ServerListenAddr, cph)
	if err != nil {
		log.Fatal("new server err", err)
	}
	if err := srv.Listen(nil); err != nil {
		log.Error(errors.ErrorStack(err))
	}
}
