package main

import (
	"encoding/json"
	"fmt"
)

type DeliveryEvent struct {
	AssetID    string `json:"asset_id"`
	Subscriber string `json:"subscriber"`
	Action     string `json:"action"`
	Attempt    int    `json:"attempt"`
}

func nextAction(event DeliveryEvent) string {
	if event.Attempt >= 3 {
		return "hold"
	}
	return "deliver"
}

func enqueueDelivery(c *Client, event DeliveryEvent) error {
	// Canonical queue call: infrai.queue.publish.
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.Publish(json.RawMessage(encoded), "delivery-"+event.AssetID+"-"+event.Subscriber)
}

func processOne(c *Client, message Message) (string, error) {
	var event DeliveryEvent
	if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
		return "", err
	}
	decision := nextAction(event)
	if err := c.Ack(message.MessageID); err != nil {
		return "", err
	}
	return fmt.Sprintf("asset=%s subscriber=%s action=%s", event.AssetID, event.Subscriber, decision), nil
}
