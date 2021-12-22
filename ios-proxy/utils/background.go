/*
* @Author: 于智明
* @Date:   2021/1/11 4:03 下午
 */
package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/DeanThompson/syncmap"
	"github.com/franela/goreq"
	"go.uber.org/zap"
	"io"
	"ctp-ios-proxy/common"
	"ctp-ios-proxy/configs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const defaultDownloadTimeout = 2 * time.Hour

type BackgroundState struct {
	Message     string      `json:"message"`
	Error       string      `json:"error"`
	Progress    interface{} `json:"progress"`
	PackageName string      `json:"packageName,omitempty"`
	Status      string      `json:"status"`
	err         error
	wg          sync.WaitGroup
}

type Background struct {
	Sm *syncmap.SyncMap
	n  int
	mu sync.Mutex
	// timer *SafeTimer
}

// Get return nil if not found
func (b *Background) Get(key string) (status *BackgroundState) {
	value, ok := b.Sm.Get(key)
	if !ok {
		return nil
	}
	status = value.(*BackgroundState)
	return status
}

func (b *Background) genKey() (key string, state *BackgroundState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.n++
	key = fmt.Sprintf("%d", b.n)
	state = &BackgroundState{}
	b.Sm.Set(key, state)
	return
}

func (b *Background) HTTPDownload(urlStr string, dst string, mode os.FileMode) (key string) {
	key, state := b.genKey()
	state.wg.Add(1)
	go func() {
		defer time.AfterFunc(5*time.Minute, func() {
			b.Sm.Delete(key)
		})

		b.Get(key).Message = "downloading"
		if err := b.doHTTPDownload(key, urlStr, dst, mode); err != nil {
			b.Get(key).Message = "http download: " + err.Error()
		} else {
			b.Get(key).Message = "downloaded"
		}
	}()
	return
}
func (b *Background) Wait(key string) error {
	state := b.Get(key)
	if state == nil {
		return errors.New("not found key: " + key)
	}
	state.wg.Wait()
	return state.err
}

// Default download timeout 30 minutes
func (b *Background) doHTTPDownload(key, urlStr, dst string, fileMode os.FileMode) (err error) {
	state := b.Get(key)
	if state == nil {
		panic("http download key invalid: " + key)
	}
	defer func() {
		state.err = err
		state.wg.Done()
	}()

	res, err := goreq.Request{
		Uri:             urlStr,
		MaxRedirects:    10,
		RedirectHeaders: true,
	}.Do()
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, err := res.Body.ToString()
		if err != nil && err == bytes.ErrTooLarge {
			return fmt.Errorf("Expected HTTP Status code: %d", res.StatusCode)
		}
		return errors.New(body)
	}

	// mkdir is not exists
	os.MkdirAll(filepath.Dir(dst), 0755)

	file, err := os.Create(dst)
	if err != nil {
		return
	}
	defer file.Close()

	var totalSize int
	fmt.Sscanf(res.Header.Get("Content-Length"), "%d", &totalSize)
	wrproxy := newDownloadProxy(file, totalSize)
	defer wrproxy.Done()
	b.Get(key).Progress = wrproxy

	// timeout here
	timer := time.AfterFunc(defaultDownloadTimeout, func() {
		res.Body.Close()
	})
	defer timer.Stop()

	_, err = io.Copy(wrproxy, res.Body)
	if err != nil {
		return
	}
	if fileMode != 0 {
		os.Chmod(dst, fileMode)
	}
	b.resign(dst)
	return
}

func (b *Background) resign(dst string) {
	ipaPath, _ := filepath.Abs(dst)
	var out bytes.Buffer
	cmd := exec.Command(configs.ConfigiOS.Env.IPASigner, "resign", "-p", "configs/Resign.mobileprovision",
		ipaPath, ipaPath)
	cmd.Stdout = &out
	cmd.Run()
	common.Log.Info("resign", zap.String("log", out.String()))
}

type downloadProxy struct {
	canceled   bool
	writer     io.Writer
	TotalSize  int    `json:"totalSize"`
	CopiedSize int    `json:"copiedSize"`
	Error      string `json:"error,omitempty"`
	wg         sync.WaitGroup
}

func newDownloadProxy(wr io.Writer, totalSize int) *downloadProxy {
	di := &downloadProxy{
		writer:    wr,
		TotalSize: totalSize,
	}
	di.wg.Add(1)
	return di
}

func (d *downloadProxy) Cancel() {
	d.canceled = true
}

func (d *downloadProxy) Write(data []byte) (int, error) {
	if d.canceled {
		return 0, errors.New("download proxy was canceled")
	}
	n, err := d.writer.Write(data)
	d.CopiedSize += n
	return n, err
}

// Should only call once
func (d *downloadProxy) Done() {
	d.wg.Done()
}

func (d *downloadProxy) Wait() {
	d.wg.Wait()
}
