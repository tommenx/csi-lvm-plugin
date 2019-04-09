package lvm

import (
	"context"

	"github.com/golang/glog"
	"github.com/pborman/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
	k8sCache *ConfigCache
}

var lvmVolumes = make(map[string]*lvmVolume)

func transVolumes2Allocation() AllocationsLVM {
	allocation := AllocationsLVM{}
	for _, v := range lvmVolumes {
		allocation.Allocation = append(allocation.Allocation, *v)
	}
	return allocation
}

func NewControllerServer(d *csicommon.CSIDriver, cache *ConfigCache) csi.ControllerServer {
	c := &controllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(d),
		k8sCache:                cache,
	}
	return c
}

// provisioner create/delete lvm image
func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.Errorf("CreateVolume: driver not support Create volume: %v", err)
		return nil, err
	}
	if len(req.Name) == 0 {
		glog.Errorf("CreateVolume:Volume name cannot be empty")
		return nil, status.Error(codes.InvalidArgument, "Volume Name cannot be empty")
	}
	if req.VolumeCapabilities == nil {
		glog.Errorf("CreateVolume: Volume Capabilities cannot be empty")
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities cannot be empty")
	}

	if _, ok := req.GetParameters()["vg"]; !ok {
		glog.Errorf("CreateVolume: error VolumeGroup from input")
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: error VolumeGroup from input")
	}
	lvmVol := &lvmVolume{}
	// if bps is null , it is set to nil
	if bps, ok := req.GetParameters()["bps"]; ok {
		lvmVol.Bps = bps
	} else {
		lvmVol.Bps = "0"
	}
	lvmVol.VolName = req.Name
	if req.GetCapacityRange() != nil {
		lvmVol.VolSize = int64(req.GetCapacityRange().GetRequiredBytes())
	} else {
		lvmVol.VolSize = 1024 * 1024 * 1024
	}
	// find by name if exsist the same
	vol, _ := getLVMVolumeByName(req.Name)
	if vol != nil {
		if vol.VolSize != lvmVol.VolSize {
			// glog.V(4).Infof("CreateVolume: exist disk %s size is different with requested for disk: exist size: %s, request size: %s", req.GetName(), vol.VolSize, lvmVol.VolSize)
			return nil, status.Errorf(codes.Internal, "disk %s size is different with requested for disk", req.GetName())
		} else {
			tmpVol := &csi.Volume{
				VolumeId:      vol.VolID,
				CapacityBytes: vol.VolSize,
				VolumeContext: req.GetParameters(),
			}
			return &csi.CreateVolumeResponse{Volume: tmpVol}, nil
		}
	}
	// create LVM image
	lvmVol.VolID = uuid.NewUUID().String()
	lvmVol.VolumeGroup = req.GetParameters()["vg"]
	err := createLVMDevice(lvmVol)
	if err != nil {
		return nil, err
	}
	// set bps
	ok, maj, min := getDeviceNum(lvmVol)
	if !ok {
		glog.V(4).Infoln("ControllerServer:can't get device number")
	} else {
		lvmVol.Maj = maj
		lvmVol.Min = min
	}
	setBps(lvmVol)
	// add to lvmvolume slice
	lvmVolumes[lvmVol.VolID] = lvmVol

	node, _ := GetNodeInfo()
	err = cs.k8sCache.Update(node)
	if err != nil {
		glog.Errorf("ControllerServer: can't update configmap of node")
	}
	allocation := transVolumes2Allocation()
	for _, v := range allocation.Allocation {
		glog.V(4).Infof("%v", v)
	}
	err = cs.k8sCache.Update(allocation)
	if err != nil {
		glog.Errorf("ControllerServer: can't update configmap of allocation")
	}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      lvmVol.VolID,
			CapacityBytes: lvmVol.VolSize,
			VolumeContext: req.GetParameters(),
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.V(4).Infof("DeleteVolumes: Starting delete volume %s", req.GetVolumeId())
	// check inputs
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.Errorf("DeleteVolume: Invaild delete volume args %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "DeleteVolume: invalid delete volume args %v", err)
	}
	// find lvmVol from lvmVols
	vol, ok := lvmVolumes[req.VolumeId]
	if !ok {
		glog.V(4).Infof("DeleteVolume: Can't find the request volumeId %s", req.VolumeId)
		return &csi.DeleteVolumeResponse{}, nil
	}
	// remove the request lv
	err := deleteLVMDevice(vol)
	if err != nil {
		glog.Errorf("DeleteVolume: Can't remove %s from %s with the path %s", vol.LvmName, vol.VolumeGroup, vol.MapperPath)
		return nil, status.Errorf(codes.Internal, "DeleteVolume: Can't remove lv %s", req.GetVolumeId())
	}
	// remove from the map
	delete(lvmVolumes, req.GetVolumeId())
	// update configmap
	node, _ := GetNodeInfo()
	err = cs.k8sCache.Update(node)
	if err != nil {
		glog.Errorf("ControllerServer: can't update configmap")
	}
	allocation := transVolumes2Allocation()
	err = cs.k8sCache.Update(allocation)
	if err != nil {
		glog.Errorf("ControllerServer: can't update configmap of allocation")
	}
	// return result
	return &csi.DeleteVolumeResponse{}, nil
}
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	glog.V(4).Infof("ControllerPublishVolume is called, do nothing by now")
	return &csi.ControllerPublishVolumeResponse{}, nil
}
