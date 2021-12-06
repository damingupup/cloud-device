package tools

import (
	"bytes"
	"context"
	cloudLog "ctp-android-proxy/moudles/log"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

type MessageType int8

const (
	CONTROL_MSG_TYPE_INJECT_KEYCODE MessageType = iota
	CONTROL_MSG_TYPE_INJECT_TEXT
	CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT
	CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT
	CONTROL_MSG_TYPE_BACK_OR_SCREEN_ON
	CONTROL_MSG_TYPE_EXPAND_NOTIFICATION_PANEL
	CONTROL_MSG_TYPE_COLLAPSE_NOTIFICATION_PANEL
	CONTROL_MSG_TYPE_GET_CLIPBOARD
	CONTROL_MSG_TYPE_SET_CLIPBOARD
	CONTROL_MSG_TYPE_SET_SCREEN_POWER_MODE
)

type PositionType struct {
	X      int32 `json:"x"`
	Y      int32 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

type Message struct {
	Msg_type MessageType `json:"msg_type"`
	// CONTROL_MSG_TYPE_INJECT_KEYCODE
	Msg_inject_keycode_action    int8  `json:"msg_inject_keycode_action"`
	Msg_inject_keycode_keycode   int32 `json:"msg_inject_keycode_keycode"`
	Msg_inject_keycode_metastate int32 `json:"msg_inject_keycode_metastate"`
	// CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT
	Msg_inject_touch_action    int8         `json:"msg_inject_touch_action"`
	Msg_inject_touch_pointerid int64        `json:"msg_inject_touch_pointerid"`
	Msg_inject_touch_position  PositionType `json:"msg_inject_touch_position"`
	Msg_inject_touch_pressure  uint16       `json:"msg_inject_touch_pressure"`
	Msg_inject_touch_buttons   int32        `json:"msg_inject_touch_buttons"`
	// CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT
	Msg_inject_scroll_position   PositionType `json:"msg_inject_scroll_position"`
	Msg_inject_scroll_horizontal int32        `json:"msg_inject_scroll_horizontal"`
	Msg_inject_scroll_vertical   int32        `json:"msg_inject_scroll_vertical"`
}

type KeycodeMessage struct {
	Msg_type                     MessageType `json:"msg_type"`
	Msg_inject_keycode_action    int8        `json:"msg_inject_keycode_action"`
	Msg_inject_keycode_keycode   int32       `json:"msg_inject_keycode_keycode"`
	Msg_inject_keycode_metastate int32       `json:"msg_inject_keycode_metastate"`
}

type TouchMessage struct {
	Msg_type                   MessageType  `json:"msg_type"`
	Msg_inject_touch_action    int8         `json:"msg_inject_touch_action"`
	Msg_inject_touch_pointerid int64        `json:"msg_inject_touch_pointerid"`
	Msg_inject_touch_position  PositionType `json:"msg_inject_touch_position"`
	Msg_inject_touch_pressure  uint16       `json:"msg_inject_touch_pressure"`
	Msg_inject_touch_buttons   int32        `json:"msg_inject_touch_buttons"`
}

type ScrollMessage struct {
	Msg_type                     MessageType  `json:"msg_type"`
	Msg_inject_scroll_position   PositionType `json:"msg_inject_scroll_position"`
	Msg_inject_scroll_horizontal int32        `json:"msg_inject_scroll_horizontal"`
	Msg_inject_scroll_vertical   int32        `json:"msg_inject_scroll_vertical"`
}

//func drainScrcpyRequests(conn *net.TCPConn, reqC chan Message, ctx context.Context) error {
func drainScrcpyRequests(conn net.Conn, reqC chan Message, ctx context.Context) error {
	for req := range reqC {
		select {
		case <-ctx.Done():
			cloudLog.Logger.Warn("退出控制程序")
			return nil
		default:
			var err error
			switch req.Msg_type {
			case CONTROL_MSG_TYPE_INJECT_KEYCODE:
				t := KeycodeMessage{
					Msg_type:                     req.Msg_type,
					Msg_inject_keycode_action:    req.Msg_inject_keycode_action,
					Msg_inject_keycode_keycode:   req.Msg_inject_keycode_keycode,
					Msg_inject_keycode_metastate: req.Msg_inject_keycode_metastate,
				}
				buf := &bytes.Buffer{}
				err = binary.Write(buf, binary.BigEndian, t)
				if err != nil {
					fmt.Printf("CONTROL_MSG_TYPE_INJECT_KEYCODE error: %s", err)
					fmt.Printf("%s", buf.Bytes())
					break
				}
				_, err = conn.Write(buf.Bytes())
			case CONTROL_MSG_TYPE_INJECT_TEXT:
			case CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT:
				var pointerid int64 = -1
				var pressure uint16 = 65535
				var buttons int32 = 1
				req.Msg_inject_touch_pointerid = pointerid
				req.Msg_inject_touch_pressure = pressure
				req.Msg_inject_touch_buttons = buttons
				position := PositionType{
					X:      req.Msg_inject_touch_position.X,
					Y:      req.Msg_inject_touch_position.Y,
					Width:  req.Msg_inject_touch_position.Width,
					Height: req.Msg_inject_touch_position.Height,
				}
				RevertPointer(&position)
				t := TouchMessage{
					Msg_type:                   req.Msg_type,
					Msg_inject_touch_action:    req.Msg_inject_touch_action,
					Msg_inject_touch_pointerid: req.Msg_inject_touch_pointerid,
					Msg_inject_touch_position:  position,
					Msg_inject_touch_pressure:  req.Msg_inject_touch_pressure,
					Msg_inject_touch_buttons:   req.Msg_inject_touch_buttons,
				}
				buf := &bytes.Buffer{}
				err = binary.Write(buf, binary.BigEndian, t)
				if err != nil {
					fmt.Printf("CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT error: %s", err)
					fmt.Printf("%s", buf.Bytes())
					break
				}
				_, err = conn.Write(buf.Bytes())
			case CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT:
				position := PositionType{
					X:      req.Msg_inject_scroll_position.X,
					Y:      req.Msg_inject_scroll_position.Y,
					Width:  req.Msg_inject_scroll_position.Width,
					Height: req.Msg_inject_scroll_position.Height,
				}
				RevertPointer(&position)
				t := ScrollMessage{
					Msg_type:                     req.Msg_type,
					Msg_inject_scroll_position:   position,
					Msg_inject_scroll_horizontal: req.Msg_inject_scroll_horizontal,
					Msg_inject_scroll_vertical:   req.Msg_inject_scroll_vertical,
				}
				buf := &bytes.Buffer{}
				err = binary.Write(buf, binary.BigEndian, t)
				if err != nil {
					fmt.Printf("CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT error: %s", err)
					fmt.Printf("%s", buf.Bytes())
					break
				}
				_, err = conn.Write(buf.Bytes())
			case CONTROL_MSG_TYPE_BACK_OR_SCREEN_ON:
			case CONTROL_MSG_TYPE_EXPAND_NOTIFICATION_PANEL:
			case CONTROL_MSG_TYPE_COLLAPSE_NOTIFICATION_PANEL:
			case CONTROL_MSG_TYPE_GET_CLIPBOARD:
			case CONTROL_MSG_TYPE_SET_CLIPBOARD:
			case CONTROL_MSG_TYPE_SET_SCREEN_POWER_MODE:
			default:
				err = errors.New("unsupported msg type")
			}
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func RevertPointer(messageType *PositionType) {
	width := messageType.Width
	height := messageType.Height
	realWidth := AndroidControl.Device.RealWidth
	realHeight := AndroidControl.Device.RealHeight
	if (width > height && realWidth < realHeight) || (width < height && realWidth > realHeight) {
		realWidth = AndroidControl.Device.RealHeight
		realHeight = AndroidControl.Device.RealWidth
	}

	x := int32(float64(messageType.X) / (float64(width) / float64(realWidth)))
	messageType.X = x
	y := int32(float64(messageType.Y) / (float64(height) / float64(realHeight)))
	messageType.Y = y
	messageType.Width = int16(realWidth)
	messageType.Height = int16(realHeight)
}
