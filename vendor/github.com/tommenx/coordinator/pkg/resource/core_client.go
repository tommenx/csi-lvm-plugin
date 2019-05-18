package resource

import (
	"github.com/tommenx/coordinator/pkg/db"
)

type CoreClient struct {
	hanlder *db.EtcdHandler
}

func New(endpoonts []string) (*CoreClient, error) {
	// endpoints := []string{"http://127.0.0.1:2379"}
	handler, err := db.NewEtcdHandler(endpoonts)
	if err != nil {
		return nil, err
	}
	return &CoreClient{hanlder: handler}, nil
}

func (c *CoreClient) Node() NodeInfoInterface {
	return newNodeInfo(c.hanlder)
}

func (c *CoreClient) Pod(ns string) PodInfoInterface {
	return NewPodInfo(c.hanlder, ns)
}

func (c *CoreClient) Executor() ExecutorInterafce {
	return NewExecutor(c.hanlder)
}
