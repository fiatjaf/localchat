package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	"gopkg.in/antage/eventsource.v1"
)

var messageStreams = cmap.New()

func messageStream(w http.ResponseWriter, r *http.Request) {
	broom, _ := base64.StdEncoding.DecodeString(mux.Vars(r)["room"])
	room := string(broom)

	var es eventsource.EventSource
	ies, ok := messageStreams.Get(room)
	if ok {
		es = ies.(eventsource.EventSource)
	} else {
		es = eventsource.New(
			&eventsource.Settings{
				Timeout:        5 * time.Second,
				CloseOnTimeout: true,
				IdleTimeout:    1 * time.Minute,
			},
			func(r *http.Request) [][]byte {
				return [][]byte{
					[]byte("X-Accel-Buffering: no"),
					[]byte("Cache-Control: no-cache"),
					[]byte("Content-Type: text/event-stream"),
					[]byte("Connection: keep-alive"),
					[]byte("Access-Control-Allow-Origin: *"),
				}
			},
		)
		messageStreams.Set(room, es)
		go func() {
			for {
				time.Sleep(25 * time.Second)
				es.SendEventMessage("", "keepalive", "")
			}
		}()
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		es.SendRetryMessage(3 * time.Second)
	}()

	es.ServeHTTP(w, r)

	// now send all past messages
	messages, err := rds.LRange("localchat:"+room, 0, -1).Result()
	if err != nil {
		es.SendEventMessage("couldn't load past messages:"+err.Error(), "error", "")
	} else {
		for _, message := range messages {
			es.SendEventMessage(message, "stored-message", "")
		}
	}
}

func newMessage(w http.ResponseWriter, r *http.Request) {
	broom, _ := base64.StdEncoding.DecodeString(mux.Vars(r)["room"])
	room := string(broom)

	defer r.Body.Close()
	bmessage, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	message := string(bmessage)

	if len(message) > 500 {
		http.Error(w, "message too long", 400)
		return
	}

	spl := strings.Split(message, "|~|")
	message = fmt.Sprintf(`["%s", "%s", %d]`, spl[0], spl[1], time.Now().Unix())

	err = rds.Eval(`
local roomkey = 'localchat:' .. KEYS[1]
local message = ARGV[1]
if redis.call('llen', roomkey) > 100 then
  redis.call('lpop', roomkey)
end
redis.call('rpush', roomkey, message)
redis.call('expire', roomkey, 3600 * 24 * 7)
return 1
    `, []string{room}, message).Err()
	if err != nil {
		log.Error().Err(err).Msg("failed to store message")
		http.Error(w, "failed to store message", 500)
		return
	}

	// dispatch message to all listeners
	var es eventsource.EventSource
	ies, ok := messageStreams.Get(room)
	if ok {
		es = ies.(eventsource.EventSource)
	} else {
		http.Error(w, "no one is listening", 501)
		return
	}
	es.SendEventMessage(message, "message", "")
}
