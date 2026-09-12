package main

import "fmt"

func main() {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	event := DeliveryEvent{AssetID: "preset-042", Subscriber: "buyer-17", Action: "content.processed", Attempt: 1}
	if err := enqueueDelivery(c, event); err != nil {
		panic(err)
	}
	messages, err := c.Consume(1, 60)
	if err != nil {
		panic(err)
	}
	for _, message := range messages {
		result, err := processOne(c, message)
		if err != nil {
			panic(err)
		}
		fmt.Println(result)
	}
}
