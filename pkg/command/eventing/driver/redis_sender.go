// Copyright 2020 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/types"

	redisParse "github.com/go-redis/redis/v8"
	"github.com/gomodule/redigo/redis"
)

type RedisEventSender struct {
	Plan    SendEventsPlan
	Address string
	// NumConsumers   string
	TLSCertificate string
	Stream         string
	// Group          string
	// PodName        string
	pool *redis.Pool
}

func NewRedisSender(plan SendEventsPlan) EventSender {
	rs := &RedisEventSender{
		Plan:           plan,
		Address:        "rediss://localhost:6379",
		TLSCertificate: "",
		Stream:         "mystream",
	}
	rs.pool = newPool(rs.Address, rs.TLSCertificate)
	// rs.logger :=logging.FromContext(ctx).Desugar().With(zap.String("stream", config.Stream)),
	//logger: logger
	return rs
}

func strptr(s string) *string {
	return &s
}

func (s RedisEventSender) Send() EventsStats {
	plan := s.Plan
	senderName := plan.senderName
	//values := map[string]string{"id": "1234668888888", "source": "323223232332909090", "type": "dev.knative.eventing.test.scaling", "timestamp": "12929299999992222"}
	source := types.URIRef{URL: url.URL{Scheme: "http", Host: "example.com", Path: "/source"}}
	timestamp := types.Timestamp{Time: time.Now()}
	//schema := types.URI{URL: url.URL{Scheme: "http", Host: "example.com", Path: "/schema"}}
	e := event.Event{
		Context: event.EventContextV1{
			Type:   "com.example.FullEvent",
			Source: source,
			ID:     "full-event",
			Time:   &timestamp,
			//DataSchema: &schema,
			Subject: strptr("topic"),
		}.AsV1(),
	}
	//if err := event.SetData("text/json", "[\"fruit\", \"orange\"]"); err != nil {
	data := []byte("[\"fruit\", \"orange\"]")
	//data := []byte("{\"a\" : \"b\"}") // {"a": "b"}
	if err := e.SetData(event.ApplicationJSON, data); err != nil {
		panic(err)
	}

	startTime := time.Now()
	errCount := 0
	eventsSentCount := 0
	var contentLenght int64 = 0
	//TODO divide events to send into chunks/batches for perf impact
	chunkSize := 1
	chunkCount := 0
	eventsToSend := int(float64(plan.eventsPerSecond) * plan.durationSeconds)
	//TODO test HTTP 2 pipelining
	//TODO test CLoudEvents batch
	for i := 0; i < eventsToSend; i++ {
		//e["id"] = strconv.Itoa(i + 1)
		e.SetID(strconv.Itoa(i + 1))
		err := s.SendEvent(e)
		// TODO check response HTTP code?
		// count successes and errors
		if err != nil {
			errCount++
		} else {
			eventsSentCount++
			//contentLenght += resp.ContentLength
		}
		chunkCount++
		if chunkCount >= chunkSize {
			chunkCount = 0
			// sleep to keep events/second goal
			//durationSoFar := time.Since(start)

		}
	}
	// sleep for remaining time to reach events/second
	endTime := time.Now()
	targetEndTime := startTime.Add(time.Duration(plan.durationSeconds) * time.Second)
	//duration := time.Since(start)
	duration := endTime.Sub(startTime)
	if endTime.Before(targetEndTime) {
		sleepDuration := targetEndTime.Sub(endTime)
		fmt.Printf("Sender %s sleeping %s\n", senderName, sleepDuration)
		time.Sleep(sleepDuration)
	}
	timeSeconds := float64(duration.Nanoseconds()) / float64(time.Second)
	stats := EventsStats{plan.senderName, eventsSentCount, timeSeconds, contentLenght, errCount}
	return stats
}

func (r *RedisEventSender) SendEvent(event cloudevents.Event) error {
	//r.logger.Info("Sending event", zap.Any("event", event))
	conn, _ := r.pool.Dial()
	defer conn.Close()

	// TODO: validate event
	fmt.Println("Sending event: ", event)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		//r.logger.Error("Cannot marshall event to bytes", zap.Error(err))
		//log.Printf("Cannot marshall event to bytes: %s", err)
		return err
	}

	var field interface{}
	if err = json.Unmarshal(eventBytes, &field); err != nil {
		//r.logger.Error("Cannot unmarshall event from bytes", zap.Error(err))
		//log.Printf("Cannot marshall event to bytes: %s", err)
		return err
	}

	args := []interface{}{r.Stream, "*"}
	args = append(args, "Full Cloud Event", field)

	response, err := conn.Do("XADD", args...)
	if err != nil {
		//r.logger.Error("Cannot write to stream", zap.Error(err))
		//log.Fatalf("Cannot write to stream: %s", err)
		return err
	}

	timestamp := strings.Split(string(response.([]uint8)), "-")[0] //timestamp in first element
	fmt.Println("Event Timestamp: ", timestamp)                    //unix time in string

	fmt.Println("Added event to the stream")
	return nil
}

func newPool(address string, tlscert string) *redis.Pool {
	opt, err := redisParse.ParseURL(address)
	if err != nil {
		panic(err)
	}
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// max number of connections
		MaxActive: 12000,
		// Dial is an application supplied function for creating and
		// configuring a connection.
		Dial: func() (redis.Conn, error) {
			var c redis.Conn
			if opt.Password != "" && tlscert != "" {
				roots := x509.NewCertPool()
				ok := roots.AppendCertsFromPEM([]byte(tlscert))
				if !ok {
					panic(err)
				}
				c, err = redis.Dial("tcp", opt.Addr,
					//redis.DialUsername(opt.Username), //username needs to be empty for successful redis connection (v8 go-redis issue)
					redis.DialPassword(opt.Password),
					redis.DialTLSConfig(&tls.Config{
						RootCAs: roots,
					}),
					redis.DialTLSSkipVerify(true),
					redis.DialUseTLS(true),
				)
				if err != nil {
					panic(err)
				}
			} else {
				c, err = redis.Dial("tcp", opt.Addr)
				if err != nil {
					panic(err)
				}
			}
			return c, err
		},
	}
}
