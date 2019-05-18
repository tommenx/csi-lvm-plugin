package isolate

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	path := "/Users/tommenx/Desktop"
	err := ioutil.WriteFile(
		filepath.Join(path, "test.txt"),
		[]byte("test write to file 2 "),
		defaultDirPerm,
	)
	if err != nil {
		log.Println(err)
	}

}
