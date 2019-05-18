package resource

import (
	"errors"

	cdpb "github.com/tommenx/pvproto/pkg/proto/coordinatorpb"
)

// scheduler send to coordinator when it need to scheduler

type ResourceType int

type ResourceUnit int

type StorageLevel int

const (
	STORAGE ResourceType = iota
	LIMIT
)

var Type = map[cdpb.StorageType]ResourceType{
	cdpb.StorageType_LIMIT:   LIMIT,
	cdpb.StorageType_STORAGE: STORAGE,
}

var ReType = map[ResourceType]cdpb.StorageType{
	LIMIT:   cdpb.StorageType_LIMIT,
	STORAGE: cdpb.StorageType_STORAGE,
}

const (
	B ResourceUnit = iota
	KB
	MB
	GB
	C
)

var Unit = map[cdpb.Unit]ResourceUnit{
	cdpb.Unit_B:  B,
	cdpb.Unit_KB: KB,
	cdpb.Unit_MB: MB,
	cdpb.Unit_GB: GB,
	cdpb.Unit_C:  C,
}

var ReUnit = map[ResourceUnit]cdpb.Unit{
	B:  cdpb.Unit_B,
	KB: cdpb.Unit_KB,
	MB: cdpb.Unit_MB,
	GB: cdpb.Unit_GB,
	C:  cdpb.Unit_C,
}

const (
	HDD StorageLevel = iota
	SSD
	NVM
)

var Level = map[cdpb.StorageLevel]StorageLevel{
	cdpb.StorageLevel_HDD: HDD,
	cdpb.StorageLevel_NVM: NVM,
	cdpb.StorageLevel_SSD: SSD,
}

var ReLeval = map[StorageLevel]cdpb.StorageLevel{
	HDD: cdpb.StorageLevel_HDD,
	SSD: cdpb.StorageLevel_SSD,
	NVM: cdpb.StorageLevel_NVM,
}

type Resource struct {
	Type ResourceType `json:"type"`
	Kind string       `json:"kind"`
	Size uint64       `json:"size"`
	Unit ResourceUnit `json:"unit"`
}

type Storage struct {
	Name      string       `json:"name"` // vg
	Level     StorageLevel `json:"level"`
	Resources []*Resource  `json:"resources"`
}

type Device struct {
	Id          string `json:"id"`
	Name        string `json:"name"` //lv name
	Maj         string `json:"maj"`
	Min         string `json:"min"`
	VolumeGroup string `json:"volume_group"` //来源于Storage中的name
	DevicePath  string `json:"device_path"`
}

type PVC struct {
	Name      string      `json:"name"` // pvc-name
	Resources []*Resource `json:"resources"`
}

type PV struct {
	Name   string  `json:"name"`
	Device *Device `json:"device"`
}

type Allocation struct {
	PVC *PVC `json:"pvc"`
	PV  *PV  `json:"pv"`
}

type Pod struct {
	Node        string        `json:"node"`
	Name        string        `json:"name"`
	Namespace   string        `json:"namespace"`
	Allocations []*Allocation `json:"allocation"`
}

type Node struct {
	Name     string     `json:"name"`
	Storages []*Storage `json:"storages"`
}

type Executor struct {
	Hostname string `json:"host_name"`
	Address  string `json:"address"`
}

var (
	ErrKeyNotExist     = errors.New("key not exist")
	ErrKeyAlreadyExist = errors.New("key already exist")
)
