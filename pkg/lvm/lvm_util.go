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
	VolName     string `json:"vol_name"`
	LvmName     string `json:"lvm_name"`
	VolID       string `json:"vol_id"`
	DevicePath  string `json:"device_path"`
	MapperPath  string `json:"mapper_path"`
	VolumeGroup string `json:"volume_group"`
	Maj         string `json:"maj"`
	Min         string `json:"min"`
	Bps         string `json:"bps"`
	VolSize     int64  `json:"volume_size"`
}

type AllocationsLVM struct {
	Allocation []lvmVolume `json:"allocation"`
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
	if ok, maj, min := getDeviceNum(lvm); ok {
		lvm.Maj = maj
		lvm.Min = min
	}
	lvm.DevicePath = fmt.Sprintf("/dev/%s/%s", lvm.VolumeGroup, lvm.LvmName)
	lvm.MapperPath = fmt.Sprintf("/dev/mapper/%s-%s", lvm.VolumeGroup, lvm.LvmName)
	glog.V(4).Infof("success create lvm [%s] in vg [%s] with the path %s", lvm.LvmName, lvm.VolumeGroup, lvm.MapperPath)
	return nil
}

//TODO
// update the /etc/fstab
func deleteLVMDevice(lvm *lvmVolume) error {
	glog.V(4).Infof("lvm: delete %s in %s ", lvm.VolName, lvm.VolumeGroup)
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

func getDeviceNum(lvm *lvmVolume) (bool, string, string) {
	label := fmt.Sprintf("%s-%s", lvm.VolumeGroup, lvm.LvmName)
	args := []string{`--output`, `NAME,MAJ:MIN`}
	out, err := execCommand("lsblk", args)
	lines := strings.Split(string(out), "\n")
	var dn string
	for _, line := range lines {
		if ok := strings.Contains(line, label); ok {
			cols := strings.Split(strings.Trim(line, " "), " ")
			dn = cols[len(cols)-1]
		}
	}
	if err != nil {
		return false, "", ""
	}
	if len(dn) == 0 {
		return false, "", ""
	}
	strs := strings.Split(dn, ":")
	return true, strs[0], strs[1]
}

func setBps(lvm *lvmVolume) error {
	cgpath := fmt.Sprintf("/sys/fs/cgroup/blkio/csi-lvm/%s/", lvm.VolName)
	args1 := []string{"-p", cgpath}
	_, err := execCommand("mkdir", args1)
	if err != nil {
		return err
	}
	// set the bps
	str := fmt.Sprintf(`"%s:%s %s"`, lvm.Maj, lvm.Min, lvm.Bps)
	writePath := cgpath + "blkio.throttle.write_bps_device"
	com := fmt.Sprintf(`echo "%s" > %s`, str, writePath)
	cmd := exec.Command("bash", "-c", com)
	fmt.Println("command is", com)
	cmd.Start()
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("error:%v", err)
		glog.V(4).Infof("SET BPS echo error , %v", err)
		return err
	}
	return nil
}
