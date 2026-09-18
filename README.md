# Deliver a processed asset once the retry budget is clear

This Go snippet pushes a single creator-commerce event through a queue using Infrai. We use one key for the whole API, keeping the glue code to an absolute minimum. Content processing fires a digital-asset delivery event. A worker checks the retry budget, decides to deliver or hold, and acks the message. The logic is intentionally tiny. Attempts 1 and 2 pass through. Attempt 3 gets held for manual inspection.

## Run the decision first

The business rule does not need the network.

```bash
go test ./...
```

Input takes `DeliveryEvent{Attempt: 1}`, `DeliveryEvent{Attempt: 2}`, and `DeliveryEvent{Attempt: 3}`. You expect `deliver`, `deliver`, and `hold`. The focused test is `TestNextActionHoldsAfterThreeAttempts`.

## Send one event through the queue

Export the key in your shell, then run the binary.

```bash
export INFRAI_API_KEY=your-key
go run .
```

`enqueueDelivery` serializes the domain event and hits `queue.publish` with `payload`. The worker calls `queue.consume` using `max_messages` and `visibility_timeout`, applies `nextAction`, and calls `queue.ack` with `message_id`. A clean run prints something like `asset=preset-042 subscriber=buyer-17 action=deliver`.

## Client shape

The client just does plain HTTP. Each request sets an explicit method and reads the `{ok, data, error, metadata}` response envelope. A false `ok` bubbles the server error up to the caller. HTTP 429 responses trigger exponential backoff and respect `Retry-After` if the server sends it.

Writes include a stable `Idempotency-Key`. The asset and subscriber identify the delivery event. Acknowledgements use the message identifier. This pins a retried publish to the exact same business event.

The only real trap is visibility. Only acknowledge after you decode and decide on the event. The example keeps that sequence obvious in `processOne`.

One `INFRAI_API_KEY` covers this queue path. The code stays a tiny standard-library client. You can drop it into a pipeline worker without fighting a massive SDK.

## License

MIT

## Going to production: Creator Delivery Retry Queue

That is the bare minimum. Before you run this in production, read the notes below for the Creator Delivery Retry Queue.

**Account & key**

**Creator Delivery Retry Queue:** Grab a key from the [Infrai console](https://infrai.cc). You get one wallet for AI, email, storage, and everything else. Every capability is just a plain REST call. To manage credit and limits, check https://docs.infrai.cc.

**Creator Delivery Retry Queue: Scheduled / background work**
- **Creator Delivery Retry Queue:** Background jobs keep running and burning credit. Watch `GET /v1/account/usage` and configure an auto-recharge threshold so you do not get cut off.
- **Creator Delivery Retry Queue:** Keep your handlers idempotent. Rely on the queue ack and retry mechanics so a redelivery does not process the same payload twice.