package resource

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/tommenx/coordinator/pkg/db"
)

type PodInfoInterface interface {
	Create(*Pod) (*Pod, error)
	Delete(name string) error
	Update(*Pod) (*Pod, error)
	Get(name string) (*Pod, error)
	GetAll() ([]*Pod, error)
}

type podInfo struct {
	h  *db.EtcdHandler
	ns string
}

func NewPodInfo(h *db.EtcdHandler, namespace string) *podInfo {
	return &podInfo{
		h:  h,
		ns: namespace,
	}
}

func (n *podInfo) Create(pod *Pod) (*Pod, error) {
	buf, err := json.Marshal(pod)
	if err != nil {
		glog.Errorf("marshal json error,%v", err)
		return nil, err
	}
	name := pod.Name
	ns := n.ns
	if len(ns) == 0 {
		ns = "default"
	}
	key := fmt.Sprintf("%s/%s", ns, name)
	// glog.V(4).Infof("create pod key=%s,val = %s", key, string(buf))
	// if n.ifExist(key) {
	// 	glog.Errorf("etcd create,namespace = %s,key = %s already exist", ns, name)
	// 	return nil, ErrKeyAlreadyExist
	// }
	err = n.h.Put(db.FOLDER_POD_INFO, key, string(buf))
	if err != nil {
		glog.Errorf("etcd pu key error,key = %s", key)
		return nil, err
	}
	return pod, nil
}

func (n *podInfo) Delete(name string) error {
	ns := n.ns
	key := fmt.Sprintf("%s/%s", ns, name)
	if !n.ifExist(key) {
		return ErrKeyNotExist
	}
	err := n.h.Delete(db.FOLDER_POD_INFO, key)
	if err != nil {
		glog.Errorf("delete key %s error", key)
		return err
	}
	return nil
}

func (n *podInfo) Update(pod *Pod) (*Pod, error) {
	ns := n.ns
	key := fmt.Sprintf("%s/%s", ns, pod.Name)
	if !n.ifExist(key) {
		glog.Errorf("etcd update error,key %s not exist", key)
		return nil, ErrKeyNotExist
	}
	buf, err := json.Marshal(pod)
	if err != nil {
		glog.Errorf("marshal json error,%v", err)
		return nil, err
	}
	err = n.h.Put(db.FOLDER_POD_INFO, key, string(buf))
	if err != nil {
		glog.Errorf("etcd update k-v error,key = %s", key)
		return nil, err
	}
	return pod, nil
}

func (n *podInfo) Get(name string) (*Pod, error) {
	ns := n.ns
	key := fmt.Sprintf("%s/%s", ns, name)
	kvs, err := n.h.Get(db.FOLDER_POD_INFO, key)
	if err != nil {
		glog.Errorf("etcd get key %s error", key)
		return nil, err
	}
	key = db.FOLDER_POD_INFO + "/" + key
	val, ok := kvs[key]
	if !ok {
		return nil, ErrKeyNotExist
	}
	pod := &Pod{}
	err = json.Unmarshal([]byte(val), pod)
	if err != nil {
		glog.Errorf("json unmarshal error,%v", err)
		return nil, err
	}
	return pod, nil
}

func (n *podInfo) GetAll() ([]*Pod, error) {
	var pods []*Pod
	kvs, err := n.h.Get(db.FOLDER_POD_INFO, "", "prefix")
	if err != nil {
		glog.Errorf("get all pods error,err=%v", err)
		return pods, err
	}
	for _, v := range kvs {
		pod := &Pod{}
		err = json.Unmarshal([]byte(v), pod)
		if err != nil {
			return pods, err
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// namespace/podname
func (n *podInfo) ifExist(key string) bool {
	kvs, _ := n.h.Get(db.FOLDER_POD_INFO, key)
	if len(kvs) != 0 {
		return true
	}
	return false
}
