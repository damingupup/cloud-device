package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"ctp-ios-proxy/utils/bpool"
	"ctp-ios-proxy/utils/errd"
	"net/http"
	"nhooyr.io/websocket"
	"time"
)

var PicCh chan []byte
var RetryTime int

//var ChList []chan []byte
//var ChList map[string]chan []byte
var TimeStamp = time.Now().Format("2006-01-02")

type RespBody struct {
	Cd    int                    `json:"cd"`
	Msg   string                 `json:"msg"`
	Data  map[string]interface{} `json:"data"`
	Total int                    `json:"total"`
}

type HttpHandler struct {
	Log      *zap.Logger
	HttpCode int
	Cd       int
	Msg      string
	Data     map[string]interface{}
	Total    int
	Ctx      *gin.Context
}

func (obj *HttpHandler) Json() {
	if obj.HttpCode == 0 {
		obj.HttpCode = http.StatusOK
	}
	body := RespBody{Cd: obj.Cd, Msg: obj.Msg, Data: obj.Data, Total: obj.Total}
	obj.Ctx.JSON(obj.HttpCode, body)
}

func init() {
	//ChList = map[string]chan []byte{"video": make(chan []byte, 1024*1024), "save": make(chan []byte, 1024*1024)}
}

type WsTool struct {
	Conn *websocket.Conn
}

func (ws *WsTool) ReadJson(ctx context.Context, v interface{}) (err error) {
	defer errd.Wrap(&err, "failed to read JSON message")
	_, r, err := ws.Conn.Reader(ctx)
	if err != nil {
		return err
	}
	b := bpool.Get()
	defer bpool.Put(b)
	_, err = b.ReadFrom(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b.Bytes(), v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

func (ws *WsTool) Write(ctx context.Context, v interface{}) error {
	return ws.write(ctx, v)
}

func (ws *WsTool) write(ctx context.Context, v interface{}) (err error) {
	defer errd.Wrap(&err, "failed to write JSON message")
	w, err := ws.Conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return err
}
