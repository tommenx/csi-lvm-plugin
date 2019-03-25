package lvm

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/tommenx/csi-lvm-plugin/pkg/util"
)

const (
	PluginFolder = "/var/lib/kubelet/pligins/lvmplugin.csi.alibabacloud.com"
	DriverName   = "lvmplugin.csi.alibabacloud.com"
	CSIVersion   = "v1.0.0"
)

type lvm struct {
	driver           *csicommon.CSIDriver
	endpoint         string
	idServer         csi.IdentityServer
	nodeServer       csi.NodeServer
	controllerServer csi.ControllerServer
	cap              []*csi.VolumeCapability_AccessMode
	cscap            []*csi.ControllerServiceCapability
}

func NewDriver(nodeID, endpoint string, cache *util.ConfigCache) *lvm {
	tmplvm := &lvm{}
	tmplvm.endpoint = endpoint
	if nodeID == "" {
		nodeID = "zx"
		glog.V(4).Infof("use default nodeID: %s", nodeID)
	}
	csiDriver := csicommon.NewCSIDriver(DriverName, CSIVersion, nodeID)
	tmplvm.driver = csiDriver
	tmplvm.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	})
	tmplvm.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	// TODO
	// create GRPC SERVER
	tmplvm.idServer = csicommon.NewDefaultIdentityServer(tmplvm.driver)
	tmplvm.controllerServer = NewControllerServer(tmplvm.driver, cache)
	tmpns, err := NewNodeServer(tmplvm.driver, false)
	if err != nil {
		glog.Errorf("lvm can't start node server,err %v \n", err)
	}
	tmplvm.nodeServer = tmpns

	return tmplvm
}

func (lvm *lvm) Run() {
	glog.V(4).Infof("Starting csi-plugin Driver: %v version: %v", DriverName, CSIVersion)
	server := csicommon.NewNonBlockingGRPCServer()
	server.Start(lvm.endpoint, lvm.idServer, lvm.controllerServer, lvm.nodeServer)
	server.Wait()
}
