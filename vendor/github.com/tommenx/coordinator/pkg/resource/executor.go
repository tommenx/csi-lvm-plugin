package resource

import (
	"github.com/tommenx/coordinator/pkg/db"
)

type ExecutorInterafce interface {
	Register(*Executor) error
	Watch(name string, adder, delter func(k, v string))
	Stop()
}

type executor struct {
	h *db.EtcdHandler
}

func NewExecutor(h *db.EtcdHandler) *executor {
	return &executor{
		h: h,
	}
}

func (n *executor) Register(info *Executor) error {
	err := n.h.Register(db.FOLDER_EXECUTOR_INFO, info.Hostname, info.Address)
	return err
}

func (n *executor) Watch(name string, adder, delter func(k, v string)) {
	n.h.Watch(db.FOLDER_EXECUTOR_INFO, "", adder, delter, "prefix")
}

func (n *executor) Stop() {
	n.h.Stop()
}
