package server

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/tommenx/coordinator/pkg/resource"
	"github.com/tommenx/csi-lvm-plugin/pkg/config"
	"github.com/tommenx/csi-lvm-plugin/pkg/isolate"
	"github.com/tommenx/csi-lvm-plugin/pkg/utils"
	cdpb "github.com/tommenx/pvproto/pkg/proto/coordinatorpb"
	ecpb "github.com/tommenx/pvproto/pkg/proto/executorpb"
	"google.golang.org/grpc"
)

type Executor struct {
	client *resource.CoreClient
	coord  cdpb.CoordinatorClient
}

type server struct{}

func New(etcds []config.Etcd, cd config.Coordinator) *Executor {
	strs := []string{}
	for _, v := range etcds {
		strs = append(strs, fmt.Sprintf("%s:%s", v.Ip, v.Port))
	}
	client, err := resource.New(strs)
	if err != nil {
		glog.Errorf("create resource client error,err=%v", err)
		panic(err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", cd.Ip, cd.Port), grpc.WithInsecure())
	if err != nil {
		glog.Errorf("create grpc client error,err=%v", err)
		panic(err)
	}
	coordinatorClient := cdpb.NewCoordinatorClient(conn)
	return &Executor{
		client: client,
		coord:  coordinatorClient,
	}
}

func (s *server) PutIsolation(ctx context.Context, req *ecpb.PutIsolationRequest) (*ecpb.PutIsolationResponse, error) {
	rsp := &ecpb.PutIsolationResponse{
		Header: &ecpb.ResponseHeader{
			Error: &ecpb.Error{},
		},
	}
	glog.V(4).Infof("get req %+v\n", req)
	device := req.Deivice
	limits := req.Resource
	settings := []*isolate.BlkioSetting{}
	for _, limit := range limits {
		if limit.Type == ecpb.StorageType_STORAGE {
			continue
		}
		setting := &isolate.BlkioSetting{
			Type:  limit.Kind,
			Maj:   device.Maj,
			Min:   device.Min,
			Count: utils.ToCount(int64(limit.Size), limit.Unit),
		}
		settings = append(settings, setting)
	}
	// use tempPath for test
	tempPath := "/Users/tommenx/Desktop/cgroup"
	err := isolate.NewBlkio(tempPath).Update(isolate.DefaultDir, settings)
	if err != nil {
		rsp.Header.Error.Type = ecpb.ErrorType_INTERNAL_ERROR
		rsp.Header.Error.Message = fmt.Sprintf("update blkio error,err=%v", err)
	}
	rsp.Header.Error.Type = ecpb.ErrorType_OK
	rsp.Header.Error.Message = "success"
	return rsp, nil

}

// 是阻塞型的因此运行的时候需要开启一个协程
func (s *Executor) Resister(l *config.Local, nodeId string) {
	exec := &resource.Executor{
		Hostname: nodeId,
		Address:  fmt.Sprintf("%s:%s", l.Ip, l.Port),
	}
	s.client.Executor().Register(exec)
}

func (s *Executor) ReportStorage(nodeId string, disks []*config.Disk) error {
	rpcNode := utils.ToRPCNode(nodeId, disks)
	req := &cdpb.PutNodeResourceRequest{
		Header: &cdpb.RequestHeader{},
		Node:   rpcNode,
	}
	rsp, err := s.coord.PutNodeResource(context.Background(), req)
	if err != nil {
		glog.Errorf("node repoort storage resource error,node=%+v,err=%+v", *rpcNode, err)
		return err
	}
	if rsp.Header.Error.Type != 0 {
		glog.Errorf("put node resource error,code=%d, msg=%s", rsp.Header.Error.Type, rsp.Header.Error.Message)
		return fmt.Errorf("put node resource error,code=%d, msg=%s", rsp.Header.Error.Type, rsp.Header.Error.Message)
	}
	return nil
}

func (s *Executor) Run(l *config.Local) {
	lst, err := net.Listen("tcp", fmt.Sprintf(":%s", l.Port))
	if err != nil {
		panic(err)
	}
	gRPC := grpc.NewServer()
	ecpb.RegisterExecutorServer(gRPC, &server{})
	gRPC.Serve(lst)
}
