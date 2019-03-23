package util

import (
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
	defaultNS  = "default"
	defaultNID = "zx"
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

func (cache *ConfigCache) Get() (*v1.ConfigMap, error) {
	resourceID := fmt.Sprintf("csi-lvm-%s", cache.NodeID)
	cm, err := cache.Client.CoreV1().ConfigMaps(cache.Namespace).Get(resourceID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// 1. check if exist
// 2. create configmap, and double check if it is exist
// 3. update the configmap data
func (cache *ConfigCache) Create(data interface{}) error {
	return nil
}
