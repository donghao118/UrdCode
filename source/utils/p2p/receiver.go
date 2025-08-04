package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Receiver struct {
	localIP   string
	localPort int

	ctx context.Context

	channelMap map[byte]Reactor

	mtx sync.Mutex
}

func NewReceiver(localIP string, localPort int, ctx context.Context) *Receiver {
	return &Receiver{
		localIP:   localIP,
		localPort: localPort,

		ctx:        ctx,
		channelMap: make(map[byte]Reactor),
		mtx:        sync.Mutex{},
	}
}

func (r *Receiver) AddChennel(reactor Reactor, channel_id byte) {
	r.channelMap[channel_id] = reactor
}

func (r *Receiver) Start() error {
	ipPort := fmt.Sprintf("%s:%d", r.localIP, r.localPort)
	// 监听指定端口
	listener, err := net.Listen("tcp", ipPort)
	if err != nil {
		return fmt.Errorf(fmt.Sprintln("Error listening:", err))
	}

	log.Println("Receiver启动，开始监听", ipPort)

	go func() {
		defer listener.Close()
		for {
			// 接受传入连接
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			go r.handleConnection(conn)
		}
	}()
	return nil
}

func (r *Receiver) handleConnection(conn net.Conn) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(30 * time.Minute)); err != nil {
		panic(err)
	}
	clientReader := bufio.NewReader(conn)
	//log.Println("start")
	for {
		// 读取消息长度
		bz, err := clientReader.ReadBytes('\n')
		if len(bz) == 0 {
			continue
		}
		if bz[len(bz)-1] == '\n' {
			bz = bz[:len(bz)-1]
		}

		select {
		case <-r.ctx.Done():
			log.Println("client closed the connection by terminating the process")
			return
		default:
		}

		switch err {
		case nil:
			var channelMessage = new(Envelop)
			if err := json.Unmarshal(bz, channelMessage); err != nil {
				log.Printf("envelop marshal error: %v\n", err)
				//log.Println(clientRequest)
				//fmt.Println(string(clientRequest))
			} else {
				chid := channelMessage.Channel_id
				if reactor, ok := r.channelMap[chid]; ok {
					if err := reactor.Receive(chid, channelMessage.GetMessage(), channelMessage.MessageType); err != nil {
						fmt.Println(err)
					}
				} else {
					log.Printf("error: 通道%c不存在\n", chid)
				}
			}
		case io.EOF:
			log.Println("client closed the connection by terminating the process")
			return
		default:
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (r *Receiver) Stop() {}
