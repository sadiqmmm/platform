// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	"testing"
	"time"

	"github.com/mattermost/platform/model"
)

func TestWebSocket(t *testing.T) {
	th := Setup().InitBasic()
	WebSocketClient, err := th.CreateWebSocketClient()
	if err != nil {
		t.Fatal(err)
	}
	defer WebSocketClient.Close()

	time.Sleep(300 * time.Millisecond)

	// Test closing and reconnecting
	WebSocketClient.Close()
	if err := WebSocketClient.Connect(); err != nil {
		t.Fatal(err)
	}

	WebSocketClient.Listen()

	time.Sleep(300 * time.Millisecond)

	WebSocketClient.SendMessage("ping", nil)
	time.Sleep(300 * time.Millisecond)
	if resp := <-WebSocketClient.ResponseChannel; resp.Data["text"].(string) != "pong" {
		t.Fatal("wrong response")
	}

	WebSocketClient.SendMessage("", nil)
	time.Sleep(300 * time.Millisecond)
	if resp := <-WebSocketClient.ResponseChannel; resp.Error.Id != "api.web_socket_router.no_action.app_error" {
		t.Fatal("should have been no action response")
	}

	WebSocketClient.SendMessage("junk", nil)
	time.Sleep(300 * time.Millisecond)
	if resp := <-WebSocketClient.ResponseChannel; resp.Error.Id != "api.web_socket_router.bad_action.app_error" {
		t.Fatal("should have been bad action response")
	}

	req := &model.WebSocketRequest{}
	req.Seq = 0
	req.Action = "ping"
	WebSocketClient.Conn.WriteJSON(req)
	time.Sleep(300 * time.Millisecond)
	if resp := <-WebSocketClient.ResponseChannel; resp.Error.Id != "api.web_socket_router.bad_seq.app_error" {
		t.Fatal("should have been bad action response")
	}

	WebSocketClient.UserTyping("", "")
	time.Sleep(300 * time.Millisecond)
	if resp := <-WebSocketClient.ResponseChannel; resp.Error.Id != "api.websocket_handler.invalid_param.app_error" {
		t.Fatal("should have been invalid param response")
	} else {
		if resp.Error.DetailedError != "" {
			t.Fatal("detailed error not cleared")
		}
	}
}

func TestWebSocketEvent(t *testing.T) {
	th := Setup().InitBasic()
	WebSocketClient, err := th.CreateWebSocketClient()
	if err != nil {
		t.Fatal(err)
	}
	defer WebSocketClient.Close()

	WebSocketClient.Listen()

	omitUser := make(map[string]bool, 1)
	omitUser["somerandomid"] = true
	evt1 := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_TYPING, "", th.BasicChannel.Id, "", omitUser)
	evt1.Add("user_id", "somerandomid")
	Publish(evt1)

	time.Sleep(300 * time.Millisecond)

	stop := make(chan bool)
	eventHit := false

	go func() {
		for {
			select {
			case resp := <-WebSocketClient.EventChannel:
				if resp.Event == model.WEBSOCKET_EVENT_TYPING && resp.Data["user_id"].(string) == "somerandomid" {
					eventHit = true
				}
			case <-stop:
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond)

	stop <- true

	if !eventHit {
		t.Fatal("did not receive typing event")
	}

	evt2 := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_TYPING, "", "somerandomid", "", nil)
	go Publish(evt2)
	time.Sleep(300 * time.Millisecond)

	eventHit = false

	go func() {
		for {
			select {
			case resp := <-WebSocketClient.EventChannel:
				if resp.Event == model.WEBSOCKET_EVENT_TYPING {
					eventHit = true
				}
			case <-stop:
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond)

	stop <- true

	if eventHit {
		t.Fatal("got typing event for bad channel id")
	}
}

func TestZZWebSocketTearDown(t *testing.T) {
	// *IMPORTANT* - Kind of hacky
	// This should be the last function in any test file
	// that calls Setup()
	// Should be in the last file too sorted by name
	time.Sleep(2 * time.Second)
	TearDown()
}
