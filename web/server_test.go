package web_test

import (
	"testing"

	"github.com/ShevaXu/golang/web"
)

func TestGetLocalIP(t *testing.T) {
	t.Log("IP:", web.GetLocalIP())
}

func TestDownloadFile(t *testing.T) {
	if err := web.DownloadFile("http://www.baidu.com", "/tmp/test-download"); err != nil {
		t.Error(err.Error())
	}
}
