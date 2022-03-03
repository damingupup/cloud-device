package utils

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"ios-proxy/common"
	"ios-proxy/configs"
	"net/http"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
)

var WDAControl WDA

type WDA struct {
	WdAUrl      string
	AgentPort   string //视频端口 9100转发
	ControlPort string // 控制端口 8100转发
	Uid         string
	CmdList     []common.CmdUtil
	Status      int
	Video       *Publisher
	Retry       int
	Code        string
	Server      *grpc.Server
	VideoConn   *http.Response
	WDAPid      int
}

func (w *WDA) Start(ctx context.Context, wg *sync.WaitGroup) {
	w.Video = CreatePublisher(time.Nanosecond, 1024)
	defer func() {
		fmt.Println("执行一次结束")
		wg.Done()
		w.Video.Close()
		if w.VideoConn != nil {
			w.VideoConn.Body.Close()
		}

	}()
	errCh := make(chan string, 10)
	w.IproxyStart()
	w.handleSteam(errCh)
	bodyTmp := ""
	for {
		select {
		case <-ctx.Done():
			return
		case errMsg := <-errCh:
			common.Log.Error(errMsg)
			return
		default:
			buf := make([]byte, 1024)
			n, err := w.VideoConn.Body.Read(buf)
			if err != nil {
				common.Log.Error(err.Error())
				w.Status = configs.StatusRestart
				w.stopWda()
				if w.Status == configs.StatusRestart && w.Status < 15 {
					common.RetryTime = 0
					time.Sleep(time.Second)
					w.Retry += 1
					w.handleSteam(errCh)
					continue
				} else {
					return
				}

			}
			recvBuf := buf[:n]
			tmp := string(recvBuf)
			bodyTmp = bodyTmp + tmp
			bodyTmp = w.handleData(bodyTmp)

		}
	}

}

func (w *WDA) handleSteam(errs chan string) {
	w.Status = configs.StatusWait
	//启动wda
	w.startWda()
	w.VideoConn = w.ping()
	if w.VideoConn == nil {
		common.Log.Error("手机视频连接出现问题")
		w.Status = configs.StatusError
		errs <- "手机视频连接出现问题"
		return
	}
	w.Status = configs.StatusSuccess
}
func (w *WDA) stopWda() {
	err := syscall.Kill(-w.WDAPid, syscall.SIGKILL)
	if err != nil && err.Error() != "no such process" {
		common.Log.Warn(err.Error())
	}
}
func (w *WDA) ping() *http.Response {
	url := "http://0.0.0.0:" + w.AgentPort
	data, err := http.Get(url)
	if err != nil {
		time.Sleep(time.Second)
		common.RetryTime += 1
		common.Log.Info("", zap.Int("retry", common.RetryTime))
		if common.RetryTime > 20 {
			data = nil
			return data
		} else {
			return w.ping()
		}
	}
	w.Retry = 0
	//wda8100可以访问后，9100有时候会有延迟
	return data
}

func (w *WDA) handleData(data string) (leftStr string) {
	symbolNum := strings.Count(data, "--BoundaryString")
	if symbolNum >= 2 {
		//如果--标志出现在数据中超过两次，则进行分割处理
		//首先使用\r\n进行分割
		firstStep := strings.SplitN(data, "\r\n\r\n", 2)
		secondStep := strings.SplitN(firstStep[1], "--BoundaryString", 2)
		w.Video.Publish([]byte(secondStep[0]))
		return w.handleData("--BoundaryString" + secondStep[1])
	} else {
		return data
	}
}

func (w *WDA) Stop() {
	for _, v := range w.CmdList {
		v.Stop()
	}
}

func (w *WDA) startWda() {
	//cmd := exec.Command("/Applications/Xcode.app/Contents/Developer/usr/bin/xcodebuild",
	//	"-project","WebDriverAgent.xcodeproj","-scheme",
	//	"WebDriverAgentRunner","-destination","id=2918d98bc6db836f6bc2657a9d37a65702a558d7","test")
	//var out buffer.Buffer
	//cmd.Stdout = &out
	//cmd.Stderr = &out
	//cmd.Dir = configs.ConfigiOS.Server.WDAPath
	//cmd.Start()
	//for  {
	//	fmt.Println(out.String())
	//	time.Sleep(time.Second*2)
	//
	//}
	args := []string{"-project", "WebDriverAgent.xcodeproj", "-scheme", "WebDriverAgentRunner", "-destination", "id=" + w.Uid, "test"}
	cmd := exec.Command(configs.ConfigiOS.Server.Xcode, args...)
	cmd.Dir = configs.ConfigiOS.Server.WDAPath
	cmd.Start()
	cmd1 := common.CmdUtil{
		Path: configs.ConfigiOS.Server.Xcode,
		Args: args,
		Dir:  configs.ConfigiOS.Server.WDAPath,
	}
	cmd1.Cmd = *cmd
	w.addCmd(cmd1)
	w.WDAPid = cmd.Process.Pid
}

func (w *WDA) IproxyStart() {
	agentCmd := common.CmdUtil{
		Path: configs.ConfigiOS.Server.Iproxy,
		Args: []string{"iproxy", "-u", w.Uid, "-s", "0.0.0.0", w.AgentPort, configs.WdaVideoPort},
	}
	agentCmd.Exec()
	w.addCmd(agentCmd)
	controlCmd := common.CmdUtil{
		Path: configs.ConfigiOS.Server.Iproxy,
		Args: []string{"iproxy", "-u", w.Uid, "-s", "0.0.0.0", w.ControlPort, configs.WdaControlPort},
	}
	controlCmd.Exec()
	w.addCmd(controlCmd)
	return
}
func (w *WDA) Restart() {
	w.startWda()
}
func (w *WDA) addCmd(cmd common.CmdUtil) {
	w.CmdList = append(w.CmdList, cmd)
}

//视频发布订阅
type (
	subscriber chan []byte //订阅者为一个管道
	filterFunc func(v interface{}) bool
)

type Publisher struct {
	sync.RWMutex                           //读写锁
	buffer       int                       //订阅队列的缓存大小
	timeout      time.Duration             //发布超时时间
	subscribers  map[subscriber]filterFunc //订阅者信息
}

func CreatePublisher(publishTimeout time.Duration, buffer int) *Publisher {
	return &Publisher{
		buffer:      buffer,
		timeout:     publishTimeout,
		subscribers: make(map[subscriber]filterFunc),
	}
}

func (p *Publisher) Subscribe() chan []byte {
	return p.SubscribeTopic(nil)
}

func (p *Publisher) SubscribeTopic(filter filterFunc) chan []byte {
	ch := make(chan []byte, 1024)
	p.Lock()
	p.subscribers[ch] = filter
	p.Unlock()
	return ch
}

func (p *Publisher) Evict(sub chan []byte) {
	p.Lock()
	defer p.Unlock()
	delete(p.subscribers, sub)
	close(sub)
}

func (p *Publisher) Publish(v []byte) {
	p.Lock()
	defer p.Unlock()
	var wg sync.WaitGroup
	for sub, filter := range p.subscribers {
		wg.Add(1)
		go p.callFilter(sub, filter, v, &wg)
	}
	wg.Wait()
}

func (p *Publisher) callFilter(sub subscriber, filter filterFunc, v []byte, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			fmt.Printf("Publisher callback err: %s", err)
			fmt.Printf("Pubblisher err stack: %s", debug.Stack())
		}
	}()
	if filter != nil && !filter(v) {
		return
	}

	select {
	case sub <- v:
	case <-time.After(p.timeout):
	}
}

func (p *Publisher) Close() {
	p.Lock()
	defer p.Unlock()
	for sub := range p.subscribers {
		delete(p.subscribers, sub)
		close(sub)
	}
}
