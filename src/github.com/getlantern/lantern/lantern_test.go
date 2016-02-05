package lantern

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"golang.org/x/net/proxy"

	"github.com/getlantern/testify/assert"
)

const expectedBody = "Google is built by a large team of engineers, designers, researchers, robots, and others in many different sites across the globe. It is updated continuously, and built with more tools and technologies than we can shake a stick at. If you'd like to help us out, see google.com/careers.\n"

type testProtector struct{}

func TestProxyingHTTP(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "testconfig")
	if assert.NoError(t, err, "Unable to create temp configDir") {
		defer os.RemoveAll(tmpDir)
		addr, _, err := Start(tmpDir, 5000)
		if assert.NoError(t, err, "Should have been able to start lantern") {
			newAddr, _, err := Start("testapp", 5000)
			if assert.NoError(t, err, "Should have been able to start lantern twice") {
				if assert.Equal(t, addr, newAddr, "2nd start should have resulted in the same address") {
					err = testProxiedRequest(newAddr, false)
					assert.NoError(t, err, "Proxying request should have worked")
				}
			}
		}
	}
}

func TestProxyingSOCKS(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "testconfig")
	if assert.NoError(t, err, "Unable to create temp configDir") {
		defer os.RemoveAll(tmpDir)
		_, addr, err := Start(tmpDir, 5000)
		if assert.NoError(t, err, "Should have been able to start lantern") {
			_, newAddr, err := Start("testapp", 5000)
			if assert.NoError(t, err, "Should have been able to start lantern twice") {
				if assert.Equal(t, addr, newAddr, "2nd start should have resulted in the same address") {
					err = testProxiedRequest(newAddr, true)
					assert.NoError(t, err, "Proxying request should have worked")
				}
			}
		}
	}
}

func testProxiedRequest(proxyAddr string, socks bool) error {
	var req *http.Request

	req = &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "www.google.com",
			Path:   "http://www.google.com/humans.txt",
		},
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Host": {"www.google.com:80"},
		},
	}

	transport := &http.Transport{}
	if socks {
		// Set up SOCKS proxy
		proxyURL, err := url.Parse("socks5://" + proxyAddr)
		if err != nil {
			return fmt.Errorf("Failed to parse proxy URL: %v\n", err)
		}

		socksDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return fmt.Errorf("Failed to obtain proxy dialer: %v\n", err)
		}
		transport.Dial = socksDialer.Dial
	} else {
		// Set up HTTP proxy
		transport.Dial = func(n, a string) (net.Conn, error) {
			//return net.Dial("tcp", "127.0.0.1:9898")
			return net.Dial("tcp", proxyAddr)
		}
	}

	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: transport,
	}

	var res *http.Response
	var err error

	if res, err = client.Do(req); err != nil {
		return err
	}

	var buf []byte

	buf, err = ioutil.ReadAll(res.Body)

	fmt.Printf(string(buf))

	if string(buf) != expectedBody {
		return errors.New("Expecting another response.")
	}

	return nil
}