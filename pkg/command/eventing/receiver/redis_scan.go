/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package receiver

import (
	"errors"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

//1) 1) "mystream"
//   2) 1) 1) 1519073278252-0
//         2) 1) "foo"
//            2) "value_1"
//      2) 1) 1519073279157-0
//         2) 1) "foo"
//            2) "value_2"

type StreamElements []StreamElement

type StreamElement struct {
	// Name is the stream name
	Name string

	// Items is the stream items (ID and list of field-value pairs)
	Items []StreamItem
}

type StreamItem struct {
	// ID is the item ID
	ID string

	// FieldValue represent the unscan list of field-value pairs
	FieldValues []string
}

func ScanXReadReply(src []interface{}, dst StreamElements) (StreamElements, error) {
	if dst == nil || len(dst) != len(src) {
		dst = make(StreamElements, len(src))
	}

	for i, stream := range src {
		elem, err := redis.Values(stream, nil)
		if err != nil {
			return nil, err
		}
		if len(elem) != 2 {
			return nil, fmt.Errorf("unexpected stream element slice length (%d)", len(elem))
		}

		name, err := redis.String(elem[0], nil)
		if err != nil {
			return nil, err
		}

		dst[i].Name = name

		items, err := redis.Values(elem[1], nil)
		if err != nil {
			return nil, err
		}

		if len(dst[i].Items) != len(items) {
			// Reallocate
			dst[i].Items = make([]StreamItem, len(items))
		}

		for j, rawitem := range items {
			item, err := redis.Values(rawitem, nil)
			if err != nil {
				return nil, err
			}

			if len(item) != 2 {
				return nil, fmt.Errorf("unexpected stream item slice length (%d)", len(elem))
			}

			id, err := redis.String(item[0], nil)
			if err != nil {
				return nil, err
			}
			dst[i].Items[j].ID = id

			fvs, err := redis.Values(item[1], nil)
			if err != nil {
				return nil, err
			}

			if len(dst[i].Items[j].FieldValues) != len(fvs) {
				// Reallocate
				dst[i].Items[j].FieldValues = make([]string, len(fvs))
			}

			for k, rawfv := range fvs {
				fv, err := redis.String(rawfv, nil)
				if err != nil {
					return nil, err
				}
				dst[i].Items[j].FieldValues[k] = fv
			}
		}
	}
	return dst, nil
}

//XINFO GROUPS mystream
//1) 1) name
//2) "mygroup"
//3) consumers
//4) (integer) 2
//5) pending
//6) (integer) 2
//7) last-delivered-id
//8) "1588152489012-0"
//1) 1) name
//2) "some-other-group"
//3) consumers
//4) (integer) 1
//5) pending
//6) (integer) 0
//7) last-delivered-id
//8) "1588152498034-0"

type StreamGroups map[string]StreamGroup

type StreamGroup struct {
	// Consumers is the number of consumers in the group
	Consumers int
	// Pending is the number of the pending messages (not ACKED)
	Pending int
	// LastDeliveredId is the ID of the last delivered item
	LastDeliveredId string
}

func ScanXInfoGroupReply(reply interface{}, err error) (StreamGroups, error) {
	if err != nil {
		return nil, err
	}
	groups, err := redis.Values(reply, nil)
	if err != nil {
		return nil, errors.New("expected a reply of type array")
	}
	dst := make(StreamGroups)

	for _, group := range groups {
		entries, err := redis.Values(group, nil)
		if err != nil {
			return nil, err
		}

		if len(entries) != 8 {
			return nil, fmt.Errorf("unexpected group reply size (%d)", len(entries))
		}

		name, err := redis.String(entries[1], nil)
		if err != nil {
			return nil, err
		}

		consumers, err := redis.Int(entries[3], nil)
		if err != nil {
			return nil, err
		}

		pending, err := redis.Int(entries[5], nil)
		if err != nil {
			return nil, err
		}

		lastid, err := redis.String(entries[7], nil)
		if err != nil {
			return nil, err
		}

		dst[name] = StreamGroup{
			Consumers:       consumers,
			Pending:         pending,
			LastDeliveredId: lastid,
		}
	}
	return dst, nil
}

//XPENDING mystream mygroup [<start-id> <end-id> <count> [<consumer-name>]]
//1) 1) 1526569498055-0
//   2) "Bob"
//   3) (integer) 74170458
//   4) (integer) 1
//2) 1) 1526569506935-0
//   2) "Bob"
//   3) (integer) 74170458
//   4) (integer) 1

type PendingMessages []PendingMessage

type PendingMessage struct {
	// MessageID is the ID of the pending message
	MessageID string
	// ConsumerName is the name of the consumer who was sent the pending message
	ConsumerName string
	// IdleTime is how much milliseconds have passed since the last time the message was delivered to the consumer
	IdleTime int
	// NumberOfDeliveries  is the number of times that this message was delivered to the consumer
	DeliveryCount int
}

func ScanXPendingReply(reply interface{}, err error) (PendingMessages, error) {
	if err != nil {
		return nil, err
	}
	pendingmessages, err := redis.Values(reply, nil)
	if err != nil {
		return nil, errors.New("expected a reply of type array")
	}
	dst := make(PendingMessages, len(pendingmessages))

	for i, pendingmessage := range pendingmessages {
		elem, err := redis.Values(pendingmessage, nil)
		if err != nil {
			return nil, err
		}

		if len(elem) != 4 {
			return nil, fmt.Errorf("unexpected pending message element slice length (%d)", len(elem))
		}

		msgid, err := redis.String(elem[0], nil)
		if err != nil {
			return nil, err
		}

		name, err := redis.String(elem[1], nil)
		if err != nil {
			return nil, err
		}

		idletime, err := redis.Int(elem[2], nil)
		if err != nil {
			return nil, err
		}

		delcount, err := redis.Int(elem[3], nil)
		if err != nil {
			return nil, err
		}

		dst[i] = PendingMessage{
			MessageID:     msgid,
			ConsumerName:  name,
			IdleTime:      idletime,
			DeliveryCount: delcount,
		}
	}
	return dst, nil
}
