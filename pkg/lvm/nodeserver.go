package lvm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"os/exec"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/tommenx/csi-lvm-plugin/pkg/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/util/nsenter"
	k8sexec "k8s.io/utils/exec"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
	mounter  mount.Interface
	executor *server.Executor
}

func NewNodeServer(d *csicommon.CSIDriver, containerized bool, executor *server.Executor) (*nodeServer, error) {
	mounter := mount.New("")
	if containerized {
		ne, err := nsenter.NewNsenter(nsenter.DefaultHostRootFsPath, k8sexec.New())
		if err != nil {
			return nil, err
		}
		mounter = mount.NewNsenterMounter("", ne)
	}
	return &nodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d),
		mounter:           mounter,
		executor:          executor,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nscap := &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			},
		},
	}
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			nscap,
		},
	}, nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	source := req.StagingTargetPath
	targetPath := req.TargetPath
	glog.V(4).Infof("NodePublishVolume: Starting mount, source %s > target %s", source, targetPath)
	if !strings.HasSuffix(targetPath, "/mount") {
		return nil, status.Errorf(codes.InvalidArgument, "malformed the value of target path: %s", targetPath)
	}
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Volume ID must be provided")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Staging Target Path must be provided")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Volume Capability must be provided")
	}
	// ensure the path exist
	if err := ns.mounter.MakeDir(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// check if it is mounted
	mounted, err := isMounted(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if mounted {
		glog.Errorf("NodePulishVolume: %s is already mounted", targetPath)
	}

	// start to mount -option bind
	mnt := req.VolumeCapability.GetMount()
	options := append(mnt.MountFlags, "bind")
	if req.Readonly {
		options = append(options, "ro")
	}
	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}
	glog.V(4).Infof("NodePublishVolume: Starting mount source %s target %s with flags %v and fsType %s", source, targetPath, options, fsType)
	// start to mount
	if err := ns.mounter.Mount(source, targetPath, fsType, options); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.V(4).Infof("NodePublishVolume: Mount Successful: target %v", targetPath)
	// report pv info to coordinator
	volumeId := req.VolumeId
	vol, _ := lvmVolumes[volumeId]
	retry := true
	err = ns.executor.ReportPVInfo(vol.VolName, vol.LvmName, vol.VolumeGroup, vol.Maj, vol.Min, vol.DevicePath, vol.VolID)
	if retry && err != nil {
		err = ns.executor.ReportPVInfo(vol.VolName, vol.LvmName, vol.VolumeGroup, vol.Maj, vol.Min, vol.DevicePath, vol.VolID)
		retry = false
	}
	if err != nil {
		glog.Errorf("report pv error,err=%v, vol=%v", err, vol)
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

// this step is to umount the lv to the target path
func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	targetPath := req.GetTargetPath()
	// check if the folder is still exist
	exist, err := ns.mounter.ExistsPath(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exist {
		glog.V(4).Infof("NodeUnpublishVolume: folder %s dosen't exist", targetPath)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}
	mnt, err := isMounted(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, "NodeUnpublishVolume: can't find mount path")
	}
	if !mnt {
		// glog.Errorf("NodeUnpublishVolume: targetpath:%s not mount volume", targetPath)
		glog.V(4).Infof("NodeUnpublishVolume: targetpath:%s not mount volume", targetPath)
		// return nil, status.Error(codes.Internal, "NodeUnpublishVolume: target path is not a mount point")
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	err = ns.mounter.Unmount(targetPath)
	if err != nil {
		glog.Errorf("NodeUnpublishVolume: can't umount the target path %s", targetPath)
		return nil, status.Error(codes.Internal, "NodeUnpublishVolume: can't umount the target path")

	}
	glog.V(4).Infof("NodeUnpublishVolume: success unmount the target path %s", targetPath)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	glog.V(4).Infof("NodeStageVolume: stage disk %s, taget path: %s", req.GetVolumeId(), req.StagingTargetPath)
	// check the input args
	targetPath := req.StagingTargetPath
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume: no volumeId is provided")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}
	// ensure the target path
	// StagingTargetPath is like /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-20769dae-2616-11e9-b900-00163e0b8d64/globalmount
	if err := ns.mounter.MakeDir(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "NodeStageVolume: can't mkdir targetPath: %s", targetPath)
	}
	// check if it is mounted
	notMnt, err := ns.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !notMnt {
		glog.Errorf("NodeStageVolume: path: %s is already mounted", targetPath)
		return nil, status.Error(codes.Internal, "NodeStageVolume: path has ready mounted")
	}
	// start to format and mount the logical volume
	vol, ok := lvmVolumes[req.VolumeId]
	if !ok {
		glog.Errorf("NodeStageVolume: can't find %s in the lvmVols", req.GetVolumeId())
		return nil, status.Error(codes.Internal, "NodeStageVolume: can't find the requiested lvmVol")
	}
	devicePath := vol.MapperPath
	mnt := req.VolumeCapability.GetMount()
	fsType := mnt.GetFsType()
	if fsType == "" {
		fsType = "ext4"
	}
	options := []string{}
	deviceMouter := &mount.SafeFormatAndMount{Interface: ns.mounter, Exec: mount.NewOsExec()}
	if err := deviceMouter.FormatAndMount(devicePath, targetPath, fsType, options); err != nil {
		glog.Errorf("node stage volume error,err=%v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	targetPath := req.GetStagingTargetPath()
	glog.V(4).Infof("NodeUnstageVolume: Starting to unstage volume,target %s", targetPath)
	// check the arguments
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume: no VolumeID provided")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume: no target path is provided")
	}
	// check the folder exsists and umont it
	exist, err := ns.mounter.ExistsPath(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// if exist,umount it
	// else ignore it
	if exist {
		notMnt, err := ns.mounter.IsLikelyNotMountPoint(targetPath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if notMnt {
			// return nil, status.Error(codes.NotFound, "NodeUnstageVolume: Volume not mounted")
			glog.V(4).Infof("NodeUnstageVolume:path: %s Volume not mounted", targetPath)
			return &csi.NodeUnstageVolumeResponse{}, nil
		}
		err = ns.mounter.Unmount(targetPath)
		if err != nil {
			glog.Errorf("NodeUnstageVolume: can't unmount %s", targetPath)
		}
	} else {
		glog.V(4).Infof("NodeUnstageVolume: folder %s not exist", targetPath)
	}
	glog.V(4).Infof("NodeStageVolume: success unstage volume")
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func isMounted(targetpath string) (bool, error) {
	if len(targetpath) == 0 {
		return false, errors.New("no target path is provided")
	}
	findmntCmd := "findmnt"
	_, err := exec.LookPath(findmntCmd)
	if err != nil {
		return false, fmt.Errorf("%s is not in the PATH", findmntCmd)
	}
	args := []string{"-o", "TARGET,PROPAGATION,FSTYPE,OPTIONS", "-m", targetpath}
	out, err := exec.Command(findmntCmd, args...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking mounted failed,err:%v and output %s", err, string(out))
	}
	strs := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(strs) > 1 {
		return true, nil
	} else {
		return false, nil
	}
}
