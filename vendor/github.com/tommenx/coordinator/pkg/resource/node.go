package resource

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"

	"github.com/tommenx/coordinator/pkg/db"
)

type NodeInfoInterface interface {
	Create(*Node) (*Node, error)
	Update(*Node) (*Node, error)
	Get(name string) (*Node, error)
	GetAll() ([]*Node, error)
	Delete(name string) error
}

// etcd handler and node name
type nodeInfo struct {
	h *db.EtcdHandler
}

func newNodeInfo(h *db.EtcdHandler) *nodeInfo {
	return &nodeInfo{
		h: h,
	}
}

func (n *nodeInfo) Create(node *Node) (*Node, error) {
	buf, err := json.Marshal(node)
	if err != nil {
		glog.Errorf("marshal json error,%v", err)
		return nil, err
	}
	nodeName := node.Name
	// if n.ifExist(nodeName) {
	// 	glog.Errorf("etcd create,key %s already exist", node.Name)
	// 	return nil, ErrKeyAlreadyExist
	// }
	err = n.h.Put(db.FOLDER_NODE_INFO, nodeName, string(buf))
	if err != nil {
		glog.Errorf("etcd put k-v error,key = %s", nodeName)
		return nil, err
	}
	return node, nil
}

func (n *nodeInfo) ifExist(nodeName string) bool {
	kvs, _ := n.h.Get(db.FOLDER_NODE_INFO, nodeName)
	if len(kvs) != 0 {
		return true
	}
	return false
}

func (n *nodeInfo) Update(new *Node) (*Node, error) {
	if !n.ifExist(new.Name) {
		glog.Errorf("etcd update error,key %s not exist", new.Name)
		return nil, ErrKeyNotExist
	}
	buf, err := json.Marshal(new)
	if err != nil {
		glog.Errorf("marshal json error,%v", err)
		return nil, err
	}
	err = n.h.Put(db.FOLDER_NODE_INFO, new.Name, string(buf))
	if err != nil {
		glog.Errorf("etcd update k-v error,key = %s", new.Name)
		return nil, err
	}
	return new, nil
}

func (n *nodeInfo) Delete(name string) error {
	// if !n.ifExist(name) {
	// 	return ErrKeyNotExist
	// }
	err := n.h.Delete(db.FOLDER_NODE_INFO, name)
	if err != nil {
		glog.Errorf("etcd delete key %s error", name)
		return err
	}
	return nil
}

func (n *nodeInfo) Get(name string) (*Node, error) {
	kvs, err := n.h.Get(db.FOLDER_NODE_INFO, name)
	if err != nil {
		glog.Errorf("etcd get key %s error", name)
		return nil, err
	}
	key := fmt.Sprintf("%s/%s", db.FOLDER_NODE_INFO, name)
	val, ok := kvs[key]
	if !ok {
		return nil, ErrKeyNotExist
	}
	node := &Node{}
	err = json.Unmarshal([]byte(val), node)
	if err != nil {
		glog.Errorf("json unmarshal error,%v", err)
		return nil, err
	}
	return node, nil
}

// have not test
func (n *nodeInfo) GetAll() ([]*Node, error) {
	var nodes []*Node
	kvs, err := n.h.Get(db.FOLDER_NODE_INFO, "", "prefix")
	if err != nil {
		glog.Errorf("etcd get node info error, err=%s", err)
		return nodes, err
	}

	for k, v := range kvs {
		node := &Node{}
		err := json.Unmarshal([]byte(v), node)
		if err != nil {
			glog.Errorf("Unmarshal node error, key=%s, err=%v\n", k, err)
			return nodes, err
		}
		// log.Printf("key=%s,val=%s\n", k, v)
		nodes = append(nodes, node)
	}
	return nodes, nil
}
