package main

import (
	"Annerpc"
	"fmt"
	"log"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := Annerpc.Register(&foo); err != nil {
		
	}
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	// start the Server
	go startServer(addr)

	client, _ := Annerpc.Dial("tcp", <-addr)
	defer func() {
		_ = client.Close()
	}()
	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("annerpc req %d", i)
			var reply string
			if err := client.Call("Foo.sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
