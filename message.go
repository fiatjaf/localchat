package main

import (
	"encoding/base64"
	"encoding/json"
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

func storedMessages(w http.ResponseWriter, r *http.Request) {
	if rds == nil {
		json.NewEncoder(w).Encode([]string{})
		return
	}

	broom, _ := base64.StdEncoding.DecodeString(mux.Vars(r)["room"])
	room := string(broom)

	messages, err := rds.LRange("localchat:"+room, 0, -1).Result()
	if err != nil {
		log.Error().Err(err).Msg("failed to load past messages")
		http.Error(w, "failed to load past messages", 511)
		return
	}

	jmessages := make([]json.RawMessage, len(messages))
	for i, message := range messages {
		jmessages[i] = json.RawMessage(message)
	}
	json.NewEncoder(w).Encode(jmessages)
}

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

	if rds != nil {
		err = rds.Eval(`
local roomkey = 'localchat:' .. KEYS[1]
local message = ARGV[1]
if redis.call('llen', roomkey) > 100 then
  redis.call('rpop', roomkey)
end
redis.call('lpush', roomkey, message)
redis.call('expire', roomkey, 3600 * 24 * 7)
return 1
    `, []string{room}, message).Err()
		if err != nil {
			log.Error().Err(err).Msg("failed to store message")
			http.Error(w, "failed to store message", 500)
			return
		}
	}

	// dispatch message to all listeners
	var es eventsource.EventSource
	ies, ok := messageStreams.Get(room)
	if ok {
		es = ies.(eventsource.EventSource)
	} else {
		http.Error(w, "no one is listening", 512)
		return
	}
	es.SendEventMessage(message, "message", "")
}
