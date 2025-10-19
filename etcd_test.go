package main

import (
	"context"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestEtcd(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
		// 链接超时时长
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to connect to etcd: %v", err)
	}

	defer func() {
		if err := cli.Close(); err != nil {
			t.Logf("Failed to close etcd client: %v", err)
		}
	}()

	// put
	resp, err := cli.Put(context.Background(), "sample_key", "sample_value")
	if err != nil {
		t.Fatalf("Failed to put key-value: %v", err)
	}
	t.Logf("Put Response: %v", resp)

	// get
	getResp, err := cli.Get(context.Background(), "sample_key")
	if err != nil {
		t.Fatalf("Failed to get key-value: %v", err)
	}
	for _, kv := range getResp.Kvs {
		t.Logf("Get Key: %s, Value: %s", kv.Key, kv.Value)
	}
}
