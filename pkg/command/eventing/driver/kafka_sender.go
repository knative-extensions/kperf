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
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	cloudeventskafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/kelseyhightower/envconfig"
)

type KafkaEventSender struct {
	Plan SendEventsPlan
	//SleepTimeSeconds float64 // simulate sending time
}

func (s KafkaEventSender) Send() EventsStats {
	plan := s.Plan
	//eventsSent := int(float64(plan.eventsPerSecond) * plan.durationSeconds)
	stats := sendEvents(plan)
	// TODO better golang idiom/conversion?
	// sleepTimeNano := int64(plan.durationSeconds * float64(time.Second))
	// time.Sleep(time.Duration(sleepTimeNano))
	// timeSeconds := plan.durationSeconds
	//stats := EventsStats{plan.senderName, eventsSent, timeSeconds, 0, 0}
	return stats
}

type envConfig struct {
	// Port on which to listen for cloudevents
	//Port int `envconfig:"PORT" default:"8080"`

	// KafkaServer URL to connect to the Kafka server.
	KafkaServer string `envconfig:"KAFKA_BOOTSTRAP_SERVERS" required:"true"`

	// Subject is the nats subject to publish cloudevents on.
	Topic string `envconfig:"KAFKA_TOPIC" required:"true"`
}

func sendEvents(plan SendEventsPlan) EventsStats {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_0_0_0
	senderName := plan.senderName

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Printf("[ERROR] Failed to process envirnment variables: %s", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// log.Printf("Using HTTP PORT=%d", env.Port)
	// httpProtocol, err := cloudeventshttp.New(cloudeventshttp.WithPort(env.Port))
	// if err != nil {
	// 	log.Fatalf("failed to create http protocol: %s", err.Error())
	// }

	log.Printf("Sending to KAFKA_BOOTSTRAP_SERVERS=%s KAFKA_TOPIC=%s", env.KafkaServer, env.Topic)
	kafkaProtocol, err := cloudeventskafka.NewSender([]string{env.KafkaServer}, saramaConfig, env.Topic)
	if err != nil {
		log.Fatalf("failed to create Kafka protcol, %s", err.Error())
	}

	defer kafkaProtocol.Close(ctx)

	// Blocking call to wait for new messages from httpProtocol
	//message, err := httpProtocol.Receive(ctx)

	startTime := time.Now()
	errCount := 0
	eventsSentCount := 0
	eventsToSend := int(float64(plan.eventsPerSecond) * plan.durationSeconds)
	log.Printf("Sending events to Kafka %d", eventsToSend)
	//TODO test HTTP 2 pipelining
	//TODO test CLoudEvents batch
	for i := 0; i < eventsToSend; i++ {

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
		e.SetID(strconv.Itoa(i + 1))
		//ifs err := event.SetData("text/json", "[\"fruit\", \"orange\"]"); err != nil {
		data := []byte("[\"fruit\", \"orange\"]")
		//data := []byte("{\"a\" : \"b\"}") // {"a": "b"}
		if err := e.SetData(event.ApplicationJSON, data); err != nil {
			panic(err)
		}

		msg := binding.ToMessage(&e)
		err = kafkaProtocol.Send(ctx, msg)
		if err != nil {
			log.Printf("Error while forwarding the message: %s", err.Error())
		}
		eventsSentCount++

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
	stats := EventsStats{plan.senderName, eventsSentCount, timeSeconds, -1, errCount}
	return stats

}
