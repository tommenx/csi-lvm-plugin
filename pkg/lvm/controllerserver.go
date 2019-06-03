package lvm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/pborman/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

var lvmVolumes = make(map[string]*lvmVolume)

func NewControllerServer(d *csicommon.CSIDriver) csi.ControllerServer {
	c := &controllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(d),
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
	// 获取storeage class中的vg信息
	if _, ok := req.GetParameters()["vg"]; !ok {
		glog.Errorf("CreateVolume: error VolumeGroup from input")
		return nil, status.Error(codes.InvalidArgument, "CreateVolume: error VolumeGroup from input")
	}
	lvmVol := &lvmVolume{}
	// PV的名称
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
	// add to lvmvolume slice
	lvmVolumes[lvmVol.VolID] = lvmVol

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
	return &csi.DeleteVolumeResponse{}, nil
}
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	// step 1 check the volume id
	// step 2 if exist,return
	// step 3 if not,create lvmVolume,add to lvmVolumes

	// volumeId == volName
	// do not specify volSize
	volumeId := req.GetVolumeId()
	_, ok := lvmVolumes[volumeId]
	if ok {
		return &csi.ControllerPublishVolumeResponse{}, nil
	}
	params := req.GetVolumeContext()
	lvm := &lvmVolume{}
	if len(params["maj"]) == 0 || len(params["min"]) == 0 {
		glog.Errorf("ControllerPublishVolume:%s don't have maj or min", volumeId)
		return nil, status.Errorf(codes.Internal, "ControllerPublishVolume:%s don't have maj or min", volumeId)
	}
	if len(params["vg"]) == 0 {
		glog.Errorf("ControllerPublishVolume:%s don't have volumegroup", volumeId)
		return nil, status.Errorf(codes.Internal, "ControllerPublishVolume:%s don't have volumegroup", volumeId)
	}
	lvm.VolumeGroup = params["vg"]
	lvm.Maj = params["maj"]
	lvm.Min = params["min"]
	// if len(params["bps"]) == 0 {
	// 	lvm.Bps = "0"
	// } else {
	// 	lvm.Bps = params["bps"]
	// }
	lvm.LvmName = volumeId
	lvm.VolID = volumeId
	lvm.DevicePath = fmt.Sprintf("/dev/%s/%s", lvm.VolumeGroup, lvm.LvmName)
	lvm.MapperPath = fmt.Sprintf("/dev/mapper/%s-%s", lvm.VolumeGroup, lvm.LvmName)
	lvmVolumes[lvm.VolID] = lvm
	return &csi.ControllerPublishVolumeResponse{}, nil

}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	// glog.V(4).Infof("ControllerUnpublishVolume is called, do nothing by now")
	volumeId := req.GetVolumeId()
	vol, ok := lvmVolumes[volumeId]
	if ok {
		if vol.VolSize == 0 {
			delete(lvmVolumes, volumeId)
		}
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}
