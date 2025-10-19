package main

import (
	"testing"
	"time"
)

func TestDiscovery(t *testing.T) {
	client, err := NewEtcdDiscovery([]string{"localhost:2379"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create etcd discovery: %v", err)
	}
	addr, err := client.GetServiceAddr("order_service")
	if err != nil {
		t.Fatalf("Failed to get service address: %v", err)
	}
	t.Logf("Discovered service address: %s", addr)
	// 通过地址与服务通信的逻辑
}
