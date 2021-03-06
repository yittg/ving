package icmp

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/yittg/ving/errors"
)

func ExampleIPing_Trace() {
	ping := NewPing()
	if err := ping.Start(context.Background()); err != nil {
		log.Fatalf("start ping error, %s", err)
	}

	addr, _ := net.ResolveIPAddr("ip", "example.com")
	ttl := 1
	for {
		if latency, from, err := ping.Trace(addr, ttl, 2*time.Second); err != nil {
			if _, ok := err.(*errors.ErrTTLExceed); !ok {
				log.Println("timeout")
				break
			}
			log.Printf("%d %+v, %+v", ttl, from, latency)
			ttl++
		} else {
			log.Printf("%d %+v, %+v", ttl, from, latency)
			break
		}
	}

	// Output:
}
