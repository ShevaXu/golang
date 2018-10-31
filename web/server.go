package web

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

// QuickResponse writes standard HTTP code & text.
func QuickResponse(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	w.Write([]byte(http.StatusText(code)))
}

// ErrResponse is mainly for handling non-GET response.
func ErrResponse(w http.ResponseWriter, err error, code int) {
	if err != nil {
		http.Error(w, err.Error(), code)
	} else {
		QuickResponse(w, code)
	}
}

// HandleResponse is mainly for handling GET response.
func HandleResponse(w http.ResponseWriter, data []byte, status int, err error) {
	if err != nil {
		http.Error(w, err.Error(), status)
	} else {
		if status != http.StatusOK {
			QuickResponse(w, status)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}
}

// ParseJSONRequest reads & parses JSON requests.
func ParseJSONRequest(r *http.Request, v interface{}) error {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}

// AssertMethod checks the request's HTTP method and
// response 403 if fails.
func AssertMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		} else {
			h(w, r)
		}
	}
}

// GetLocalIP tries to check the IP of local machine.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// DownloadFile downloads a file from the url.
func DownloadFile(url, file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
