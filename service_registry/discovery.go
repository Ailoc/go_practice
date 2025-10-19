package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discovery interface {
	GetServiceAddr(name string) (string, error)
	// 监控服务的地址变化
	WatchService(name string) (<-chan string, error)
}

type DiscoveryEtcd struct {
	client *clientv3.Client
}

func NewEtcdDiscovery(endpoints []string, dialTimeout time.Duration) (*DiscoveryEtcd, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("etcd endpoints cannot be empty")
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &DiscoveryEtcd{
		client: cli,
	}, nil
}

func (d *DiscoveryEtcd) GetServiceAddr(name string) (string, error) {
	// etcd 获取服务地址逻辑
	resp, err := d.client.Get(context.Background(), name, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", errors.New("service not found")
	}
	// 随机返回一个服务地址
	randIndex := rand.Intn(len(resp.Kvs))
	addr := string(resp.Kvs[randIndex].Value)
	return addr, nil
}
