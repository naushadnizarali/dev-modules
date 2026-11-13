package redis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	moduleRedis "github.com/naushadnizarali/dev-modules/redis"
)

func TestOpenAndUseRedis(t *testing.T) {
	server := miniredis.RunT(t)
	cfg := moduleRedis.DefaultConfig()
	cfg.Addr = server.Addr()
	cfg.Protocol = 2

	client, err := moduleRedis.Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})

	if err := client.Set(context.Background(), "ready", "yes", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if got, err := client.Get(context.Background(), "ready").Result(); err != nil || got != "yes" {
		t.Fatalf("Get() = %q, %v", got, err)
	}
}

func TestOpenValidatesConfig(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		cfg  moduleRedis.Config
	}{
		{name: "nil context", cfg: moduleRedis.Config{Addr: "localhost:6379"}},
		{name: "missing address", ctx: context.Background()},
		{name: "negative DB", ctx: context.Background(), cfg: moduleRedis.Config{Addr: "localhost:6379", DB: -1}},
		{name: "invalid protocol", ctx: context.Background(), cfg: moduleRedis.Config{Addr: "localhost:6379", Protocol: 4}},
		{name: "invalid pool", ctx: context.Background(), cfg: moduleRedis.Config{Addr: "localhost:6379", PoolSize: 1, MinIdleConns: 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := moduleRedis.Open(test.ctx, test.cfg); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}
