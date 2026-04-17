package foostash_test

import (
	"context"
	"fmt"
	"os"
	"time"

	foostash "github.com/Omotolani98/foostash-go-sdk"
)

func ExampleClient_Pull() {
	client, err := foostash.New(foostash.Config{
		ServerURL:  "https://foostash.example.com",
		Project:    "payments-api",
		Env:        "prod",
		SSHKeyPath: "/var/run/secrets/foostash-key",
		MasterKey:  os.Getenv("FOOSTASH_MASTER_KEY"),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secrets, err := client.Pull(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(secrets["DB_URL"])
}

func ExampleClient_Watch() {
	client, err := foostash.New(foostash.Config{
		ServerURL:  "https://foostash.example.com",
		Project:    "payments-api",
		Env:        "prod",
		SSHKeyPath: "/var/run/secrets/foostash-key",
		MasterKey:  os.Getenv("FOOSTASH_MASTER_KEY"),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := client.Watch(ctx, 30*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	for snap := range ch {
		fmt.Printf("%d secrets at %s\n", len(snap.Secrets), snap.PulledAt.Format(time.RFC3339))
	}
}
