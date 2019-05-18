package isolate

import "fmt"

const (
	Blkio          = "blkio"
	defaultDirPerm = 0755
	DefaultDir     = "csi-lvm"
	CgroupPath     = "/sys/fs/cgroup/"
)

// unit 1KB
type BlkioSetting struct {
	Type  string
	Id    string
	Maj   uint64
	Min   uint64
	Count int64
}

func (t *BlkioSetting) format() []byte {
	return []byte(fmt.Sprintf("%d:%d %d", t.Maj, t.Min, t.Count))
}
