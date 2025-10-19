package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const LeaseTTL = 5 // 租约时间5秒
// 服务信息接口
type Service interface {
	Name() string
	Addr() string
}

// 服务注册的通用接口
type Registry interface {
	// 注册服务
	Registry(service Service) error
	// 注销服务
	DeRegistry() error
}

type RegistryEtcd struct {
	client   *clientv3.Client
	leaseID  clientv3.LeaseID
	leaseTTL int64
	// LeaseKeepAliveResponse wraps the protobuf message LeaseKeepAliveResponse.
	// type LeaseKeepAliveResponse struct {
	// 	*pb.ResponseHeader
	// 	ID  LeaseID
	// 	TTL int64
	// }
	leaseKeepAliveRespCh <-chan *clientv3.LeaseKeepAliveResponse
}

func (r *RegistryEtcd) Registry(service Service) error {
	// etcd注册逻辑
	// 申请租约
	grantResp, err := r.client.Grant(context.Background(), r.leaseTTL)
	if err != nil {
		return err
	}
	r.leaseID = grantResp.ID
	serviceName := service.Name() + "-" + uuid.New().String()
	// 注册服务并绑定租约
	_, err = r.client.Put(context.Background(), serviceName, service.Addr(), clientv3.WithLease(r.leaseID))
	if err != nil {
		return err
	}
	// 启动续约
	/*
			时间轴：  0s      1.6s     3.2s     4.8s     6.4s
		         │        │        │        │        │
		         │        ↓        ↓        ↓        ↓
		Grant    │     续约请求  续约请求  续约请求  续约请求
		创建租约  │        │        │        │        │
		TTL=5s   │        ↓        ↓        ↓        ↓
		         │     收到响应  收到响应  收到响应  收到响应
		         │        │        │        │        │
		         └────────┴────────┴────────┴────────┴────> 持续运行...
	*/
	// 底层启动一个goroutine，自动续约
	// 客户端 ─────> [LeaseKeepAlive RPC] ─────> etcd 服务器
	//            携带: LeaseID = 1234567890
	// etcd 服务器 ─────> [LeaseKeepAliveResponse] ─────> 客户端
	//               返回:
	//               {
	//                   ID: 1234567890,    // 租约 ID
	//                   TTL: 5,            // 剩余生存时间(秒)
	//               }
	r.leaseKeepAliveRespCh, err = r.client.KeepAlive(context.Background(), r.leaseID)
	if err != nil {
		return err
	}

	// 启动续约监听 goroutine
	go func() {
		// 处理续约响应
		for resp := range r.leaseKeepAliveRespCh {
			_ = resp
		}
	}()

	return nil
}
func (r *RegistryEtcd) DeRegistry() error {
	// etcd注销逻辑
	// 停止续约
	if _, err := r.client.Revoke(context.Background(), r.leaseID); err != nil {
		return err
	}

	// 关闭客户端连接
	if err := r.client.Close(); err != nil {
		return err
	}
	return nil
}

func NewEtcdRegistry(endpoints []string, timeout time.Duration, leaseTTL int64) (*RegistryEtcd, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("etcd endpoints cannot be empty")
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return &RegistryEtcd{
		client:   cli,
		leaseTTL: leaseTTL,
	}, nil
}
