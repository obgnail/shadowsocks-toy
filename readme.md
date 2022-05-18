## shadowsocks 玩具

usage:

```go
package main

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/obgnail/shadowsocks-toy/cipher"
	"github.com/obgnail/shadowsocks-toy/client"
	_ "github.com/obgnail/shadowsocks-toy/logger"
	"github.com/obgnail/shadowsocks-toy/ruleset"
	"github.com/obgnail/shadowsocks-toy/server"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"time"
)

const (
	ClientListenAddr = "127.0.0.1:4444"
	ServerListenAddr = "127.0.0.1:5555"
)

func main() {
	cph := cipher.NewByteMapCipher()
	srv, err := server.New(ServerListenAddr, cph)
	if err != nil {
		log.Fatal("new server err", err)
	}
	go startServer(srv)

	go func() {
		time.Sleep(time.Second)
		startChrome()
	}()

	//clt, err := client.New(ClientListenAddr, ServerListenAddr, cph, &ruleset.Direct{})
	clt, err := client.New(ClientListenAddr, ServerListenAddr, cph,  &ruleset.Global{})
	if err != nil {
		log.Fatal("new client err", err)
	}
	startClient(clt)
}

func startChrome() {
	chromeDriver := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	url := "https://www.baidu.com"

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
	}
	fmt.Println(string(data[:600]), "\n...")
}

func startServer(server *server.Server) {
	if err := server.Listen(nil); err != nil {
		log.Error(errors.ErrorStack(err))
	}
}

func startClient(client *client.Client) {
	if err := client.Listen(nil); err != nil {
		log.Error(errors.ErrorStack(err))
	}
}
```

## sequenceDiagram

```mermaid
sequenceDiagram
		autonumber
		participant Chrome
		participant Client
		participant Server
		participant Target
		
		Note over Chrome, Target: 准备阶段
		Server ->> Server: 监听端口
		Client ->> Client: 监听端口
		
		Note over Chrome, Target: 代理阶段
		Chrome ->> +Client: curl URL proxy:socks5://XXX(ChromeConn)
		Client -->> Chrome: sock5 protocol handshake response
		Note right of Client: sock5的校验流程(步骤13-16同)
		Chrome ->> Client: sock5 protocol request
		Client -->> Chrome: sock5 protocol request response
		Client ->> Client: 记录handshake，request ReceivedData,获取目标IP
		Client ->> Client: match ruleset
		alt match failed
        Client ->> Target: 无需通过Server，直连Target
        Target -->> Client: 返回TargetConn
        Client ->> Client: join ChromeConn and targetConn
        Note right of Client: ChromeConn,TargetConn皆是明文
    else match pass 
    		Client ->> Client: Encrypt handshake and request ReceivedData
    		Client ->> +Server: send Encrypt handshake Data(ClientConn)
    		Server -->> Client: Encrypted sock5 protocol handshake response
    		Note left of Client: 解密来自Server的数据,加密发给Server
    		Note right of Server: 解密来自Client的数据,加密发给Client
    		Client ->> Server: send Encrypt request Data
    		Server -->> Client: Encrypted sock5 protocol request response
    		Client ->> -Client: Tunnel ChromeConn and ClientConn
    		Note right of Client: ChromeConn是明文,ClientConn是密文
    		Server ->> Server: 获取目标IP
    		Server ->> Target: 连接Target
    		Target -->> Server: 返回Targetconn
    		Server ->> -Server: Tunnel ClientConn and TargetConn
    		Note right of Server: ClientConn是密文,TargetConn是明文
    end
```

## Conn

```mermaid
sequenceDiagram
		autonumber
		participant Chrome
		participant Client
		participant GFW
		participant Server
		participant Target
		
		Note over Chrome, Client: sock5明文
		Note over Client, Server: 密文
		Note over Server, Target: sock5明文
```

