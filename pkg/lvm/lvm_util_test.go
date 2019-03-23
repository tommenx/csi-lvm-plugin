package lvm

import (
	"fmt"
	"testing"
)

func TestExtractLVMName(t *testing.T) {
	tests := []struct {
		v1  string
		v2  string
		res bool
	}{
		{
			`Logical volume "lvol1" created.`,
			"lvol1",
			true,
		},
		{
			`Logical volume "lvol1" created.`,
			"lovl2",
			false,
		},
	}
	for _, v := range tests {
		res := v.v2 == extractLVMName(v.v1)
		// fmt.Println(v.v2, extractLVMName(v.v1), res)
		if res != v.res {
			t.Error(extractLVMName(v.v1), v.v2)
		}
	}
}

func TestCreateLVMDevice(t *testing.T) {
	lvm := &lvmVolume{}
	lvm.VolSize = 1024 * 1024 * 500
	lvm.VolumeGroup = "vgdata"
	createLVMDevice(lvm)
	fmt.Printf("success create lvm [%s] in vg [%s] with the path %s", lvm.LvmName, lvm.VolumeGroup, lvm.MapperPath)
}

func TestDeleteLVMDevice(t *testing.T) {
	lvm := &lvmVolume{}
	lvm.LvmName = "lov1"
	lvm.MapperPath = "/dev/mapper/vgdata-lov1"
	deleteLVMDevice(lvm)
}

func TestGetNodeInfo(t *testing.T) {
	node, _ := getNodeInfo()
	fmt.Printf("%v", node)
}
