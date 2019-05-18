package db

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"context"

	"github.com/golang/glog"
	v3 "go.etcd.io/etcd/clientv3"
)

// TODO
// context

type EtcdHandler struct {
	client  *v3.Client
	ctx     context.Context
	leaseId v3.LeaseID
	stop    chan error
}

const (
	FOLDER_NODE_INFO     = "/node"
	FOLDER_POD_INFO      = "/pod"
	FOLDER_EXECUTOR_INFO = "/executor"
)

var (
	DefaultTimeout = 5 * time.Second
)

func NewEtcdHandler(endpoints []string) (*EtcdHandler, error) {
	c, err := v3.New(v3.Config{
		Endpoints:   endpoints,
		DialTimeout: DefaultTimeout,
	})
	if err != nil {
		fmt.Println("connect failed, err:", err)
		return nil, err
	}
	h := &EtcdHandler{
		client: c,
		stop:   make(chan error),
	}
	return h, nil
}

func (h *EtcdHandler) Put(folder, key, val string) error {
	path := fmt.Sprintf("%s/%s", folder, key)
	_, err := h.client.Put(context.Background(), path, val)
	if err != nil {
		log.Printf("etcd put error,key = %s,error = %v", key, err)
		return err
	}
	return nil
}

func (h *EtcdHandler) Delete(folder, key string, strs ...string) error {
	path := fmt.Sprintf("%s/%s", folder, key)
	var opts []v3.OpOption
	for _, v := range strs {
		if v == "prefix" {
			opts = append(opts, v3.WithPrefix())
		}
	}
	_, err := h.client.Delete(context.Background(), path, opts...)
	if err != nil {
		log.Printf("etcd put error,key = %s,error = %v", key, err)
		glog.Errorf("etcd put error,key = %s,error = %v", key, err)
		return err
	}
	return nil
}

func (h *EtcdHandler) Get(folder, key string, strs ...string) (map[string]string, error) {
	kvs := make(map[string]string)
	path := fmt.Sprintf("%s/%s", folder, key)
	var opts []v3.OpOption
	for _, v := range strs {
		if v == "prefix" {
			opts = append(opts, v3.WithPrefix())
		}
	}
	res, err := h.client.Get(context.Background(), path, opts...)
	if err != nil {
		log.Fatalf("etcd get key error,key = %s,error = %v", key, err)
		return kvs, err
	}
	for _, ev := range res.Kvs {
		kvs[string(ev.Key)] = string(ev.Value)
	}
	return kvs, nil
}

func (h *EtcdHandler) Watch(folder, key string, addFunc func(k, v string), delFunc func(k, v string), strs ...string) {
	var opts []v3.OpOption
	for _, v := range strs {
		if v == "prefix" {
			opts = append(opts, v3.WithPrefix())
		}
	}
	path := fmt.Sprintf("%s/%s", folder, key)
	for {
		rch := h.client.Watch(context.Background(), path, opts...)
		for wresp := range rch {
			for _, ev := range wresp.Events {
				glog.V(4).Infof("event:%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				// log.Printf("event:%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				k := string(ev.Kv.Key)
				v := string(ev.Kv.Value)
				k = k[strings.LastIndex(k, "/")+1:]
				switch ev.Type {
				case v3.EventTypePut:
					addFunc(k, v)
				case v3.EventTypeDelete:
					delFunc(k, v)
				}
			}
		}
	}
}

func (h *EtcdHandler) Register(folder, key, value string) error {
	path := fmt.Sprintf("%s/%s", folder, key)
	ch, err := h.keepAlive(path, value)
	if err != nil {
		return err
	}
	for {
		select {
		case e := <-h.stop:
			h.revoke(key)
			return e
		case <-h.client.Ctx().Done():
			return errors.New("server closed")
		case _, ok := <-ch:
			if !ok {
				glog.V(4).Infoln("etcd keep alive channel closed")
				h.revoke(key)
				return nil
			}
		}
	}
}

func (h *EtcdHandler) Stop() {
	h.stop <- nil
}

func (h *EtcdHandler) keepAlive(key, val string) (<-chan *v3.LeaseKeepAliveResponse, error) {
	resp, err := h.client.Grant(context.TODO(), 5)
	if err != nil {
		glog.Errorf("etcd grant lease error,%v", err)
		return nil, err
	}
	_, err = h.client.Put(context.TODO(), key, val, v3.WithLease(resp.ID))
	if err != nil {
		glog.Errorf("etcd put service error,%v", err)
		return nil, err
	}
	h.leaseId = resp.ID
	return h.client.KeepAlive(context.TODO(), resp.ID)
}

func (h *EtcdHandler) revoke(svc string) error {
	_, err := h.client.Revoke(context.TODO(), h.leaseId)
	if err != nil {
		glog.Errorf("etcd revoke error,%v", err)
		return err
	}
	log.Printf("etcd successful revoke service %s\n,and leaseId = %v", svc, h.leaseId)
	glog.V(4).Infof("etcd successful revoke service %s", svc)
	return nil
}

func (h *EtcdHandler) Close() {
	h.client.Close()
}
