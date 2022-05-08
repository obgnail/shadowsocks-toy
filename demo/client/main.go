package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	"github.com/obgnail/shadowsocks-toy/client"
	_ "github.com/obgnail/shadowsocks-toy/logger"
	"github.com/obgnail/shadowsocks-toy/ruleset"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"time"
)

const (
	ClientListenAddr = "127.0.0.1:4444"
	ServerListenAddr = "127.0.0.1:5555"
)

func main() {
	cph := cipher.NewNopCipher()

	clt, err := client.New(ClientListenAddr, ServerListenAddr, cph,&ruleset.Direct{})
	//clt, err := client.New(ClientListenAddr, ServerListenAddr, cph,&ruleset.Global{})
	if err != nil {
		log.Fatal("new client err", err)
	}

	go func() {
		time.Sleep(time.Second)
		startChrome()
	}()

	if err := clt.Listen(nil); err != nil {
		log.Error(errors.ErrorStack(err))
	}
}

func startChrome() {
	chromeDriver := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	//url := "https://www.chromestatus.com/"
	url := "https://newurl02.xyz/sjhs"

	args := []string{
		"--headless",
		"--disable-gpu",
		"--dump-dom",
		"--proxy-server" + "=" + fmt.Sprintf("socks5://%s", ClientListenAddr),
		url,
	}
	cmd := exec.Command(chromeDriver, args...)
	data, err := cmd.Output()
	if err != nil {
		log.Fatal("err:", err)
		os.Exit(1)
	}
	fmt.Println(string(data[:600]))
}
