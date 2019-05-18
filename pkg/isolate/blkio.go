package isolate

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type blkioController struct {
	root string
}

func NewBlkio(root string) *blkioController {
	return &blkioController{
		root: filepath.Join(root, string(Blkio)),
	}
}

func (c *blkioController) Update(path string, settings []*BlkioSetting) error {
	return c.Create(path, settings)
}

func (c *blkioController) Path(path string) string {
	return filepath.Join(c.root, path)
}

// /cgroup/blkio/csi-lvm/volumeid/blkio.throttle.xxxx
// crete path = csi-lvm
func (c *blkioController) Create(path string, settings []*BlkioSetting) error {
	for _, v := range settings {
		devicePath := filepath.Join(c.Path(path), v.Id)
		log.Printf("path=%s", devicePath)
		if err := os.MkdirAll(devicePath, defaultDirPerm); err != nil {
			return err
		}
		if err := ioutil.WriteFile(
			filepath.Join(c.Path(path), fmt.Sprintf("blkio.%s", v.Type)),
			v.format(),
			defaultDirPerm,
		); err != nil {
			return err
		}
	}
	return nil
}
