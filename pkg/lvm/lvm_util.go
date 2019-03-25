package lvm

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

const (
	MBSIZE = 1024 * 1024
	GBSIZE = 1024 * 1024 * 1024
)

// using auto lvm name
type lvmVolume struct {
	VolName     string
	LvmName     string
	VolID       string
	DevicePath  string
	MapperPath  string
	VolumeGroup string
	VolSize     int64
}

type NodeLVMInfo struct {
	Report []struct {
		Vg []struct {
			VgName    string `json:"vg_name"`
			PvCount   string `json:"pv_count"`
			LvCount   string `json:"lv_count"`
			SnapCount string `json:"snap_count"`
			VgAttr    string `json:"vg_attr"`
			VgSize    string `json:"vg_size"`
			VgFree    string `json:"vg_free"`
		} `json:"vg"`
	} `json:"report"`
}

// TODO
// Write to the /etc/fstab to avoid host restart
func createLVMDevice(lvm *lvmVolume) error {
	var sz int
	var sz_unit string
	// MB SIZE
	if lvm.VolSize/GBSIZE <= 0 {
		sz = int(math.Ceil(float64(lvm.VolSize * 1.0 / MBSIZE)))
		sz_unit = "M"
	} else {
		sz = int(math.Ceil(float64(lvm.VolSize * 1.0 / GBSIZE)))
		sz_unit = "G"
	}
	volSz := fmt.Sprintf("%d%s", sz, sz_unit)
	// output, err := execCommand("lvcreate", []string{"-L", volSz, "-n", lvm.VolName, lvm.VolumeGroup})
	output, err := execCommand("lvcreate", []string{"-L", volSz, lvm.VolumeGroup})
	if err != nil {
		glog.Errorf("%v failed to create lvm,output: %s", err, string(output))
		return err
	}
	lvm.LvmName = extractLVMName(string(output))
	lvm.DevicePath = fmt.Sprintf("/dev/%s/%s", lvm.VolumeGroup, lvm.LvmName)
	lvm.MapperPath = fmt.Sprintf("/dev/mapper/%s-%s", lvm.VolumeGroup, lvm.LvmName)
	glog.V(4).Infof("success create lvm [%s] in vg [%s] with the path %s", lvm.LvmName, lvm.VolumeGroup, lvm.MapperPath)
	return nil
}

//TODO
// update the /etc/fstab
func deleteLVMDevice(lvm *lvmVolume) error {
	glog.V(4).Infof("lvm: delete % in %s ", lvm.VolName, lvm.VolumeGroup)
	args := []string{"-y", lvm.MapperPath}
	out, err := execCommand("lvremove", args)
	// out, err := testConfig("lvremove", args)
	if err != nil {
		glog.Errorf("%v failed to remove lvm, output: %s", err, string(out))
	}
	glog.V(4).Infof("success remove lvm [%s] in vg [%s] with the path %s", lvm.LvmName, lvm.VolumeGroup, lvm.MapperPath)
	return nil
}

func execCommand(command string, args []string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	return cmd.CombinedOutput()
}

// Logical volume "lvol1" created.
func extractLVMName(str string) string {
	strs := strings.Split(str, `"`)
	return strs[1]
}

func getLVMVolumeByName(volName string) (*lvmVolume, error) {
	for _, v := range lvmVolumes {
		if v.VolName == volName {
			return v, nil
		}
	}
	return nil, fmt.Errorf("can't find volName %s", volName)
}

// test cmd
func testConfig(cmd string, args []string) ([]byte, error) {
	for _, v := range args {
		cmd += " " + v
	}
	fmt.Println(cmd)
	return []byte(`Logical volume "lvol1" created.`), nil
}

// TODO
// get all vg in the node
// vg pv lv vgsize vgfree

func GetNodeInfo() (*NodeLVMInfo, error) {
	node := &NodeLVMInfo{}
	args := []string{"--columns", "--reportformat", "json"}
	out, err := execCommand("vgdisplay", args)
	// out := []byte(`{
	// 	"report": [
	// 		{
	// 			"vg": [
	// 				{"vg_name":"centos", "pv_count":"1", "lv_count":"3", "snap_count":"0", "vg_attr":"wz--n-", "vg_size":"231.88g", "vg_free":"4.00m"},
	// 				{"vg_name":"dock", "pv_count":"1", "lv_count":"1", "snap_count":"0", "vg_attr":"wz--n-", "vg_size":"<100.00g", "vg_free":"<90.00g"},
	// 				{"vg_name":"vgdata", "pv_count":"2", "lv_count":"1", "snap_count":"0", "vg_attr":"wz--n-", "vg_size":"381.46g", "vg_free":"379.46g"}
	// 			]
	// 		}
	// 	]
	// }`)
	// var err error
	// err = nil
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, node)
	if err != nil {
		return nil, err
	}
	for i := range node.Report[0].Vg {
		if node.Report[0].Vg[i].VgName == "centos" {
			node.Report[0].Vg = append(node.Report[0].Vg[:i], node.Report[0].Vg[i+1:]...)
			break
		}
	}
	return node, nil
}
