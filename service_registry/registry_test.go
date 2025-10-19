package main

import (
	"log"
	"os"
	"os/signal"
	"testing"
	"time"
)

type OrderService struct { // 应该是server
	name string
	addr string
}

func (o *OrderService) Name() string {
	return o.name
}
func (o *OrderService) Addr() string {
	return o.addr
}

func TestRegistry(t *testing.T) { // 实际在main函数中运行
	registry, err := NewEtcdRegistry([]string{"localhost:2379"}, 5*time.Second, LeaseTTL)
	if err != nil {
		log.Fatalf("Failed to create etcd registry: %v", err)
	}
	service1 := &OrderService{
		name: "order_service",
		addr: "localhost:8080",
	}
	service2 := &OrderService{
		name: "order_service",
		addr: "localhost:8080",
	}
	err = registry.Registry(service1)
	if err != nil {
		log.Fatalf("Failed to register service1: %v", err)
	}
	log.Printf("Service %s registered at %s", service1.Name(), service1.Addr())

	err = registry.Registry(service2)
	if err != nil {
		log.Fatalf("Failed to register service2: %v", err)
	}
	log.Printf("Service %s registered at %s", service2.Name(), service2.Addr())

	ChInt := make(chan os.Signal, 1)
	signal.Notify(ChInt, os.Interrupt)
	<-ChInt // ❌ 永远等待，直到手动按 Ctrl+C
	if err := registry.DeRegistry(); err != nil {
		log.Fatalf("Failed to deregister services: %v", err)
	}
}
