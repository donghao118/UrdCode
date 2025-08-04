package test

import (
	"context"
	"emulator/utils/p2p"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 延迟等同于ping延迟
// 1000条消息处理用时23ms，尚处于可接受范围内

var (
	receiveMtx     = sync.Mutex{}
	receiveCounter = 0
)

func TestMain(t *testing.T) {
	ip2 := "127.0.0.1"
	port2 := 26602
	peer, _ := p2p.NewPeer(fmt.Sprintf("%s:%d", ip2, port2), map[string]bool{"testchain": true}, "", 1)
	reactor := &myReactor{}

	client := p2p.NewSender(fmt.Sprintf("%s:%d", "127.0.0.1", 26603))
	client.AddPeer(peer)
	client2 := p2p.NewSender(fmt.Sprintf("%s:%d", "127.0.0.1", 26604))
	client2.AddPeer(peer)

	server := p2p.NewReceiver(ip2, port2, context.Background())
	server.AddChennel(reactor, 0x01)
	go server.Start()

	time.Sleep(2 * time.Second)

	fmt.Println(client.Start())
	fmt.Println(client2.Start())

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			client.Send(peer, 0x01, []byte(strings.Repeat("a", 256)+"\n"+strings.Repeat("a", 256)), 0)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			client2.Send(peer, 0x01, []byte(strings.Repeat("a", 256)+"\n"+strings.Repeat("a", 256)), 0)
		}
	}()

	wg.Wait()
	time.Sleep(10 * time.Second)
	fmt.Println(receiveCounter)

	client.Stop()
	server.Stop()

}

type myReactor struct{}

var _ p2p.Reactor = (*myReactor)(nil)

func (r *myReactor) AddPeer(p2p.Peer) error { return nil }
func (r *myReactor) Receive(chID byte, message []byte, messageType uint32) error {
	receiveMtx.Lock()
	defer receiveMtx.Unlock()
	receiveCounter++
	if receiveCounter%1000 == 0 {
		fmt.Println(receiveCounter, time.Now(), "====", string(message))
	}
	return nil
}
