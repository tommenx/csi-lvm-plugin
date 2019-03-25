package util

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ConfigCache struct {
	Client    *k8s.Clientset
	Namespace string
	NodeID    string
}

const (
	defaultNS    = "default"
	defaultNID   = "zx"
	cmLabel      = "createdBy"
	cmLabelValue = "lvm-csi"
)

func GetNodeID() string {
	nid := os.Getenv("NodeID")
	if len(nid) == 0 {
		nid = defaultNID
	}
	return nid

}
func NewConfigCache() *ConfigCache {
	ns := GetNamespace()
	clientset := NewK8sClient()
	nid := GetNodeID()
	return &ConfigCache{clientset, ns, nid}
}

func GetNamespace() string {
	namespace := os.Getenv("NAMESPACE")
	if len(namespace) == 0 {
		namespace = defaultNS
	}
	return namespace
}

// create a kubernetes client
func NewK8sClient() *k8s.Clientset {
	var cfg *rest.Config
	var err error
	// cPath := os.Getenv("KUBERNETES_CONFIG_PATH")
	cPath := "/root/.kube/config"
	if cPath != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", cPath)
		if err != nil {
			glog.Errorf("Failed to get cluster config with error: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			glog.Errorf("Failed to get cluster config with error: %v\n", err)
			os.Exit(1)
		}
	}
	client, err := k8s.NewForConfig(cfg)
	if err != nil {
		glog.Errorf("Failed to create client with error: %v\n", err)
		os.Exit(1)
	}
	return client
}

// get a configmap by given the name
func (cache *ConfigCache) Get() (*v1.ConfigMap, error) {
	resourceID := fmt.Sprintf("csi-lvm-%s", cache.NodeID)
	cm, err := cache.Client.CoreV1().ConfigMaps(cache.Namespace).Get(resourceID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// create configmap if it not exist
// 1. check if exist
// 2. create configmap, and double check if it is exist
// 3. update the configmap data
func (cache *ConfigCache) Create(data interface{}) error {
	identifier := fmt.Sprintf("csi-lvm-%s", cache.NodeID)
	cm, err := cache.Get()
	if cm != nil && err == nil {
		glog.V(4).Infof("configmap %s already exist", identifier)
		return nil
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("configmap convert to json error")
	}
	cm = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identifier,
			Namespace: cache.Namespace,
			Labels: map[string]string{
				cmLabel: cmLabelValue,
			},
		},
		Data: map[string]string{},
	}
	cm.Data["node"] = string(jsonStr)
	_, err = cache.Client.CoreV1().ConfigMaps(cache.Namespace).Create(cm)
	if err != nil {
		return fmt.Errorf("configmap create error %v", err)
	}
	return nil
}

// update config map
func (cache *ConfigCache) Update(data interface{}) error {
	identifier := fmt.Sprintf("csi-lvm-%s", cache.NodeID)
	cm, err := cache.Get()
	if err != nil {
		glog.Errorf("Configmap update error,%v", err)
		return err
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		glog.Errorf("Configmap convert to JSON error,%v", err)
		return err
	}
	cm.Data["node"] = string(jsonStr)
	_, err = cache.Client.CoreV1().ConfigMaps(cache.Namespace).Update(cm)
	if err != nil {
		glog.Error("Configmap create error")
		return err
	}
	glog.V(4).Infof("update configmap %s success", identifier)
	return nil
}
