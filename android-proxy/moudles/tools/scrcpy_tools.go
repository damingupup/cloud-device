package tools

import (
	"context"
	"ctp-android-proxy/configs"
	"ctp-android-proxy/global"
	cloudLog "ctp-android-proxy/moudles/log"
	"ctp-android-proxy/moudles/shell"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

type device struct {
	RealWidth     int
	RealHeight    int
	VirtualWidth  int
	VirtualHeight int
}

type TouchRequest struct {
	Operation    string  `json:"operation"`
	Index        int     `json:"index"`
	PercentX     float64 `json:"xP"`
	PercentY     float64 `json:"yP"`
	Milliseconds int     `json:"milliseconds"`
	Pressure     float64 `json:"pressure"`
}

var AndroidControl AndroidEngine

func GetFreePort() (port int) {
	listener, _ := net.Listen("tcp4", "0.0.0.0:0")
	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return
}

type AndroidEngine struct {
	WdAUrl        string
	AgentPort     string //视频端口
	ControlPort   string // 控制端口 安卓AgentPort，ControlPort相同
	Uid           string
	Status        int
	Video         *Publisher
	Retry         int
	ControlStream chan Message
	Device        device
	RemoteDebug   *shell.Service
	RemotePort    string
	Code          string
	Server        *grpc.Server

	videoConn   net.Conn
	controlConn net.Conn
}

func (a *AndroidEngine) Start(ctx context.Context, wtg *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		fmt.Println("执行完毕一次")
		wtg.Done()
		cancel()
	}()
	a.Device = device{}
	a.Video = CreatePublisher(time.Nanosecond, 1024)
	defer a.Video.Close()
	a.Status = global.StatusWait
	adb := shell.Adb{Uid: a.Uid, Wait: true}
	errCh := make(chan string, 10)
	var err error
	defer func() {
		cloudLog.Logger.Info("关闭视频流与控制流")
		if a.videoConn != nil {
			err = a.videoConn.Close()
			if err != nil {
				cloudLog.Logger.Warn(err.Error())
			}
		}
		if a.controlConn != nil {
			err = a.controlConn.Close()
			if err != nil {
				cloudLog.Logger.Warn(err.Error())
			}
		}

	}()
	a.handleStream(ctx, errCh)
	var n int
loop:
	for {
		select {
		case <-ctx.Done():
			cloudLog.Logger.Warn("退出视频流程序")
			adb.RemoveForward(AndroidControl.AgentPort)
			a.Status = global.StatusError
			return
		case errMsg := <-errCh:
			cloudLog.Logger.Error(errMsg)
			return
		default:
			if a.Status == global.StatusError {
				cloudLog.Logger.Error("启动失败", zap.String("video", "scrcpy启动失败"))
				return
			}
			body := make([]byte, 4)
			n, err = a.videoConn.Read(body)
			if err != nil {
				a.Status = global.StatusRestart
				cloudLog.Logger.Error(err.Error())
				a.Retry += 1
				a.handleStream(ctx, errCh)
				continue loop
			}
			length := 0
			for i, v := range body {
				length += int(v) << (i * 8)
			}
			realLength := 0
			var picData []byte
			for {
				if length == realLength {
					break
				}
				body = make([]byte, length)
				n, err = a.videoConn.Read(body)
				tmp := body[:n]
				picData = append(picData, tmp...)
				if err != nil {
					a.Status = global.StatusRestart
					cloudLog.Logger.Error(err.Error())
					a.Retry += 1
					a.handleStream(ctx, errCh)
					continue loop
				}
				realLength += n
				if n != length {
					length = length - n
					continue
				} else {
					break
				}
			}
			if picData[0] != 0xFF || picData[1] != 0xD8 {
				continue
			}
			if length < 100 {
				continue
			}
			a.Video.Publish(picData)
		}

	}
}

func (a *AndroidEngine) handleStream(ctx context.Context, errs chan string) {
	if a.Retry > global.PingTime {
		cloudLog.Logger.Error("重启次数过多")
		errs <- "重启次数过多"
		return
	}
	a.StopEngine()
	go a.startEngine()
	var err error
	adb := shell.Adb{Uid: a.Uid, Wait: true}
	if a.Status != global.StatusRestart {
		err = adb.Forward(a.AgentPort, global.ScrcpyPort)
		if err != nil {
			cloudLog.Logger.Error(err.Error(), zap.String("forward", "端口转发失败"))
			a.Status = global.StatusError
			errs <- err.Error()
			return
		}
	}
	host := "localhost:" + a.AgentPort
	var header = make([]byte, 2)
	for i := 0; i < 15; i++ {
		time.Sleep(time.Second)
		cloudLog.Logger.Info("重试次数", zap.Int("retry", i))
		a.videoConn, err = net.Dial("tcp", host)
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			continue
		}
		a.controlConn, err = net.Dial("tcp", host)
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			continue
		}
		a.Status = global.StatusSuccess

		_, err = a.videoConn.Read(header)
		if err != nil {
			continue
		}
		break
	}
	a.ControlStream = make(chan Message, 1000)
	go func() { drainScrcpyRequests(a.controlConn, a.ControlStream, ctx) }()
	body := make([]byte, header[1]-2)
	_, err = a.videoConn.Read(body)
	for i, v := range body {
		if i >= 4 && i <= 7 {
			a.Device.RealWidth += int(v) << ((i - 4) * 8)
		} else if i >= 8 && i <= 11 {
			a.Device.RealHeight += int(v) << ((i - 8) * 8)
		} else if i >= 12 && i <= 15 {
			a.Device.VirtualWidth += int(v) << ((i - 12) * 8)
		} else if i >= 16 && i <= 19 {
			a.Device.VirtualHeight += int(v) << ((i - 16) * 8)
		}
	}
	if err != nil {
		a.Status = global.StatusRestart
		cloudLog.Logger.Error(err.Error())
		a.Retry += 1
		a.handleStream(ctx, errs)
		return
	}
	a.Retry = 0
	return
}

func (a *AndroidEngine) startEngine() {
	//判断架构类型
	cloudLog.Logger.Info("准备启动")
	a.Status = global.StatusWait
	var err error
	args := []string{"-s", a.Uid, "shell", "getprop", "ro.product.cpu.abi"}
	apiCmd := shell.Service{Name: configs.CloudConfig.Server.AdbPath, Args: args}
	apiCmd.Run()
	api := apiCmd.Stdout.String()
	var sourcePath string
	if strings.Contains(api, "arm64-v8a") {
		//推送v8文件
		sourcePath = path.Join("./local", "arm64-v8a")
	} else {
		//推送v7文件
		sourcePath = path.Join("./local", "armeabi-v7a")
	}
	libCompress := path.Join(sourcePath, "libcompress.so")
	libTurboJpeg := path.Join(sourcePath, "libturbojpeg.so")
	scrcpyPath := path.Join("local", "scrcpy-server.jar")
	dst := "/data/local/tmp"
	adb := shell.Adb{Uid: a.Uid, Wait: true}
	adb.Shell([]string{"rm", "-rf", "/sdcard/cloud"})
	//adb.Push("./local/C.apk", "/data/local/tmp")
	//adb.Shell([]string{"pm", "install", "/data/local/tmp/C.apk"})
	//adb.Shell([]string{"pm", "disable-user", "com.android.settings"})
	err = adb.Push(libCompress, dst)
	if err != nil {
		cloudLog.Logger.Error(adb.Result())
		cloudLog.Logger.Error(err.Error(), zap.String("file", "推送libCompress失败"))
	}
	err = adb.Push(libTurboJpeg, dst)
	if err != nil {
		cloudLog.Logger.Error(err.Error(), zap.String("file", "推送libTurboJpeg失败"))
	}
	err = adb.Push(scrcpyPath, dst)
	if err != nil {
		cloudLog.Logger.Error(err.Error(), zap.String("file", "推送scrcpy失败"))
	}
	err = adb.Shell([]string{"chmod", "777", "/data/local/tmp/scrcpy-server.jar"})
	//判断版本号，安卓5以下需要单独指令
	err = adb.GetProp("ro.build.version.release")
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
	version := adb.Result()
	version = strings.TrimSpace(version)
	versions := strings.Split(version, ".")
	versionIndex, err := strconv.Atoi(string(versions[0]))
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		versionIndex = 9
	}
	/*
		adb shell CLASSPATH=/data/local/tmp/scrcpy-server.jar app_process / com.genymobile.scrcpy.Server -L
		# LD_LIBRARY_PATH的值为上步-L的返回值加:/data/local/tmp（注意有个英文冒号）
		adb shell LD_LIBRARY_PATH=???:/data/local/tmp CLASSPATH=/data/local/tmp/scrcpy-server.jar app_process / com.genymobile.scrcpy.Server
	*/
	if versionIndex >= 5 {
		//大于安卓5启动
		err = adb.Shell([]string{"CLASSPATH=/data/local/tmp/scrcpy-server.jar",
			"app_process", "/", "com.genymobile.scrcpy.Server", "-L",
		})
		if err != nil {
			cloudLog.Logger.Error(err.Error(), zap.String("CLASSPATH", "error"))
			a.Status = global.StatusError
			return
		}
		classPath := adb.Result()
		classPath = strings.TrimSpace(classPath)
		libraryPath := strings.Join([]string{"LD_LIBRARY_PATH=", classPath, ":/data/local/tmp"}, "")
		err = adb.Shell([]string{libraryPath, "CLASSPATH=/data/local/tmp/scrcpy-server.jar", "app_process", "/",
			"com.genymobile.scrcpy.Server", "-r", "24", "-P", "480", "-Q", "60"})
		if err != nil {
			cloudLog.Logger.Error(err.Error(), zap.String("start", "error"))
			a.Status = global.StatusError
			return
		}
	} else {
		//小于安卓5启动
		//adb shell mkdir -p /data/local/tmp/dalvik-cache
		//adb shell ANDROID_DATA=/data/local/tmp CLASSPATH=/data/local/tmp/scrcpy-server.jar app_process / com.genymobile.scrcpy.Server -L
		//# LD_LIBRARY_PATH的值为上步-L的返回值加:/data/local/tmp（注意有个英文冒号）
		//adb shell ANDROID_DATA=/data/local/tmp LD_LIBRARY_PATH=???:/data/local/tmp CLASSPATH=/data/local/tmp/scrcpy-server.jar app_process / com.genymobile.scrcpy.Server
		err = adb.Shell([]string{"mkdir", "-p", "/data/local/tmp/dalvik-cache"})
		if err != nil {
			a.Status = global.StatusError
			return
		}
		err = adb.Shell([]string{"ANDROID_DATA=/data/local/tmp", "CLASSPATH=/data/local/tmp/scrcpy-server.jar", "app_process", "/", "com.genymobile.scrcpy.Server", "-L"})
		if err != nil {
			a.Status = global.StatusError
			return
		}
		classPath := adb.Result()
		classPath = strings.TrimSpace(classPath)
		adb.Wait = true
		err = adb.Shell([]string{"ANDROID_DATA=/data/local/tmp", "LD_LIBRARY_PATH=" + classPath + ":/data/local/tmp", "CLASSPATH=/data/local/tmp/scrcpy-server.jar", "app_process", "/",
			"com.genymobile.scrcpy.Server", "-r", "24", "-P", "480", "-Q", "60"})
		if err != nil {
			a.Status = global.StatusError
			return
		}
	}
	adb.Stop()
	a.Status = global.StatusError
}

func (a *AndroidEngine) StopEngine() {
	adb := shell.Adb{Uid: a.Uid, Wait: true}
	//连续ping scrcpy服务两次服务就会退出
	adb.Shell([]string{"curl", "--connect-timeout", "1", "-m", "1", "http://localhost:6612"})
	adb.Stop()
	adb.Shell([]string{"curl", "--connect-timeout", "1", "-m", "1", "http://localhost:6612"})
	adb.Stop()
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
