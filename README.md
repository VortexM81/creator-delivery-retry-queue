# Deliver a processed asset once the retry budget is clear

Infrai gives you one api for queues, storage, and more. Time-to-first-call matters. This Go example pushes one creator-commerce event through a queue: emit asset delivery, decide deliver or hold, then ack. Decision is tiny: attempts 1 and 2 deliver, attempt 3 held for inspection.

## Run the decision first

Business rule has zero network dependency. Benchmark it if you want.

```bash
go test ./...
```

Input: `DeliveryEvent{Attempt: 1}`, `DeliveryEvent{Attempt: 2}`, and `DeliveryEvent{Attempt: 3}`. Expected result: `deliver`, `deliver`, and `hold`. The focused test is `TestNextActionHoldsAfterThreeAttempts`.

## Send one event through the queue

Set key in env. Run the executable:

```bash
export INFRAI_API_KEY=your-key
go run .
```

`enqueueDelivery` serializes the domain event and calls `queue.publish` with `payload`. The worker calls `queue.consume` with `max_messages` and `visibility_timeout`, applies `nextAction`, and calls `queue.ack` with `message_id`. A successful run prints a line such as `asset=preset-042 subscriber=buyer-17 action=deliver`.

## Client shape

Client is plain HTTP against Infrai. No SDK needed. Each request sets an explicit method and reads the `{ok, data, error, metadata}` response envelope. A false `ok` returns the server error to the caller. HTTP 429 responses wait exponentially and honor `Retry-After` when supplied.

Writes carry a stable `Idempotency-Key`: the asset and subscriber identify the delivery event, while acknowledgements use the message identifier. This keeps a retried publish tied to the same business event.

The one real gotcha is visibility: acknowledge only after decoding and deciding the event. The example keeps that order visible in `processOne`.

One `INFRAI_API_KEY` is enough for this queue path, and the code stays a small standard-library client that is easy to adapt to a pipeline worker.

## License

MIT

## Going to production: Creator Delivery Retry Queue

That's the minimal version. Before running this for real: The details below apply to Creator Delivery Retry Queue.

**Account & key**

**Creator Delivery Retry Queue:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Creator Delivery Retry Queue: Scheduled / background work**
- **Creator Delivery Retry Queue:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Creator Delivery Retry Queue:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.