package utils

import (
	"github.com/tommenx/csi-lvm-plugin/pkg/config"
	cdpb "github.com/tommenx/pvproto/pkg/proto/coordinatorpb"
	ecpb "github.com/tommenx/pvproto/pkg/proto/executorpb"
)

func ToCount(num int64, unit ecpb.Unit) int64 {
	var KB, MB, GB int64
	KB = 1 << 10
	MB = KB << 10
	GB = MB << 10
	if unit == ecpb.Unit_B {
		return num
	} else if unit == ecpb.Unit_KB {
		return num * KB
	} else if unit == ecpb.Unit_MB {
		return num * MB
	} else if unit == ecpb.Unit_GB {
		return num * GB
	}
	return 0
}

func ToRPCNode(nodeId string, disks []config.Disk) *cdpb.Node {
	storages := []*cdpb.Storage{}
	for _, disk := range disks {
		resources := []*cdpb.Resource{}
		for _, resource := range disk.Resouce {
			resources = append(resources, &cdpb.Resource{
				Type: Type[resource.Type],
				Kind: resource.Kind,
				Size: uint64(resource.Size),
				Unit: Unit[resource.Unit],
			})
		}
		storage := &cdpb.Storage{
			Name:     disk.Name,
			Level:    Level[disk.Level],
			Resource: resources,
		}
		storages = append(storages, storage)
	}
	return &cdpb.Node{
		Name:    nodeId,
		Storage: storages,
	}
}

var Level = map[string]cdpb.StorageLevel{
	"ssd": cdpb.StorageLevel_SSD,
	"hdd": cdpb.StorageLevel_HDD,
	"nvm": cdpb.StorageLevel_NVM,
}

var Type = map[string]cdpb.StorageType{
	"storage": cdpb.StorageType_STORAGE,
	"limit":   cdpb.StorageType_LIMIT,
}

var Unit = map[string]cdpb.Unit{
	"MB": cdpb.Unit_MB,
	"B":  cdpb.Unit_B,
	"GB": cdpb.Unit_GB,
	"KB": cdpb.Unit_KB,
	"C":  cdpb.Unit_C,
}
