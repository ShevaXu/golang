package sys_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShevaXu/golang/assert"
	"github.com/ShevaXu/golang/sys"
)

const (
	currentDir = "."
	testDir    = ".."
)

func TestReadDir(t *testing.T) {
	a := assert.NewAssert(t)

	list1, err1 := sys.ReadDir(testDir)
	list2, err2 := ioutil.ReadDir(testDir)
	a.Equal(err2, err1, "Same error if any")

	bs1, _ := json.Marshal(list1)
	bs2, _ := json.Marshal(list2)
	// a.Equal(list2, list1, "Same list") // different memory addresses
	a.Equal(string(bs2), string(bs1), "Same list")

	// t.Log(err2, string(bs2))
}

func listRecursive(name string, walkImpl func(root string, walkFn filepath.WalkFunc) error) ([]string, error) {
	var fileList []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Add the path and it's info to the file list.
		fileList = append(fileList, path)
		return nil
	}

	return fileList, walkImpl(name, walkFn)
}

func TestWalk(t *testing.T) {
	a := assert.NewAssert(t)

	list1, err1 := listRecursive(testDir, sys.Walk)
	list2, err2 := listRecursive(testDir, filepath.Walk)
	a.Equal(err2, err1, "Same error if any")

	// TODO: need determined way to test result
	_ = list1
	_ = list2

	// NOTE: this case passes on local file-system
	// bs1, _ := json.Marshal(list1)
	// bs2, _ := json.Marshal(list2)
	// a.Equal(list2, list1, "Same list") // different memory addresses
	// a.Equal(bs2, bs1, "Same list")

	// t.Log(err2, string(bs2))
}
