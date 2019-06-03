package lvm

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/tommenx/csi-lvm-plugin/pkg/server"
)

const (
	PluginFolder = "/var/lib/kubelet/pligins/lvmplugin.csi.alibabacloud.com"
	DriverName   = "lvmplugin.csi.alibabacloud.com"
	CSIVersion   = "v1.0.0"
)

type lvmDriver struct {
	driver           *csicommon.CSIDriver
	endpoint         string
	idServer         csi.IdentityServer
	nodeServer       csi.NodeServer
	controllerServer csi.ControllerServer
	cap              []*csi.VolumeCapability_AccessMode
	cscap            []*csi.ControllerServiceCapability
}

func NewDriver(nodeID, endpoint string, executor *server.Executor) *lvmDriver {
	driver := &lvmDriver{}
	driver.endpoint = endpoint
	if nodeID == "" {
		nodeID = "zx"
		glog.V(4).Infof("use default nodeID: %s", nodeID)
	}
	csiDriver := csicommon.NewCSIDriver(DriverName, CSIVersion, nodeID)
	driver.driver = csiDriver
	driver.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	})
	driver.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	driver.idServer = csicommon.NewDefaultIdentityServer(driver.driver)
	driver.controllerServer = NewControllerServer(driver.driver)
	nodeServer, err := NewNodeServer(driver.driver, false, executor)
	if err != nil {
		glog.Errorf("lvm can't start node server,err %v \n", err)
	}
	driver.nodeServer = nodeServer

	return driver
}

func (d *lvmDriver) Run() {
	glog.V(4).Infof("Starting csi-plugin Driver: %v version: %v", DriverName, CSIVersion)
	server := csicommon.NewNonBlockingGRPCServer()
	server.Start(d.endpoint, d.idServer, d.controllerServer, d.nodeServer)
	server.Wait()
}
