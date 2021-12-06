package adbclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type AdbClient struct {
	conn   net.Conn
	Uid    string
	Result chan string
	status bool
}

func (a *AdbClient) transport(uid string) error {
	var err error
	a.conn, err = net.Dial("tcp", "127.0.0.1:5037")
	if err != nil {
		return err
	}
	cmd := "host:transport:" + uid
	data := a.format(cmd)
	_, err = a.conn.Write([]byte(data))
	if err != nil {
		return err
	}
	check := a.checkOk()
	if check == nil {
		return check
	}
	header := make([]byte, 4)
	_, err = a.conn.Read(header)
	if err != nil {
		return err
	}
	length, err := strconv.ParseInt(string(header), 16, 32)
	if err != nil {
		return err
	}
	body := make([]byte, length)
	_, err = a.conn.Read(body)
	if err != nil {
		return err
	}
	if check != nil {
		return errors.New(string(body))
	}
	return nil
}
func (a *AdbClient) Shell(ctx context.Context, cmd string) error {
	a.status = false
	err := a.transport(a.Uid)
	if err != nil {
		return err
	}
	cmd = "shell:" + cmd
	cmd = a.format(cmd)
	_, err = a.conn.Write([]byte(cmd))
	if err != nil {
		return err
	}
	a.checkOk()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if a.status == true {
				close(a.Result)
				return nil
			}
			body := make([]byte, 1024*1024*2)
			var n int
			n, err = a.conn.Read(body)
			if err != nil {
				close(a.Result)
				if err.Error() == "EOF" {
					return nil
				}
				return err
			}
			a.Result <- string(body[:n])
		}
	}
}
func (a *AdbClient) readHeader(header []byte) (length int64, err error) {
	data := make([]byte, 4)
	_, err = a.conn.Read(data)
	if err != nil {
		return 0, err
	}
	length, err = strconv.ParseInt(string(header), 16, 32)
	return length, err
}

func (a *AdbClient) checkOk() error {
	data := make([]byte, 4)
	_, err := a.conn.Read(data)
	if err != nil {
		return err
	}
	if string(data) != "OKAY" {
		return errors.New("执行失败")
	}
	return nil
}

func (a *AdbClient) format(data string) string {
	data = fmt.Sprintf("%04x%s", len(data), data)
	return data
}

func (a *AdbClient) Stop() {
	a.status = true
	a.conn.Close()

}
