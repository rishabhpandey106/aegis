package main

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

func runNATSListener() {
	// Connect to the local NATS server
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Subscribe to the analytics logs subject
	_, err = nc.Subscribe("analytics.logs", func(m *nats.Msg) {
		fmt.Printf("\n[NATS MESSAGE RECEIVED]\n%s\n", string(m.Data))
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Println("🎧 Listening for logs on NATS subject 'analytics.logs'...")
	fmt.Println("Send a curl request to your proxy to see the log appear here instantly!")

	// Keep the main thread alive to continue receiving messages
	select {}
}
