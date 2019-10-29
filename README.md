# Broadcast

[![GoDoc](https://godoc.org/github.com/kyfk/broadcast?status.svg)](https://godoc.org/github.com/kyfk/broadcast)
[![Build Status](https://cloud.drone.io/api/badges/kyfk/broadcast/status.svg)](https://cloud.drone.io/kyfk/broadcast)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyfk/broadcast)](https://goreportcard.com/report/github.com/kyfk/broadcast)
[![codecov](https://codecov.io/gh/kyfk/broadcast/branch/master/graph/badge.svg)](https://codecov.io/gh/kyfk/broadcast)
[![codebeat badge](https://codebeat.co/badges/3aff2433-415b-491e-95c6-215e56b9d199)](https://codebeat.co/projects/github-com-kyfk-broadcast-master)

Broadcaster provides simplest way of broadcasting in Golang.

## Example

```go
type subscriber struct {
    id string
}
func (s *subscriber) GetID() broadcast.ID {
    return s.id
}
func (s *subscriber) Subscribe() error {
    fmt.Println("subscribe:", s.id)
    return nil
}
func (s *subscriber) Unsubscribe() error {
    fmt.Println("unsubscribe:", s.id)
    return nil
}

func main() {
    hub := broadcast.New(
        broadcast.Concurrency(10),
        broadcast.AllowDuplicableID(true),
    )
    hub.Subscribe(&subscriber{id: "id1"})
    hub.Subscribe(&subscriber{id: "id1"})
    hub.Subscribe(&subscriber{id: "id2"})
    hub.Subscribe(&subscriber{id: "id3"})

    hub.Publish(func (s Subscriber) error {
        fmt.Println("publish:", s.(*subscriber).id)
        return nil
    })

    hub.PublishTo(func (s Subscriber) error {
        fmt.Println("publish to:", s.(*subscriber).id)
        return nil
    }, "id1", "id3")

    hub.Terminate()
}

// Output:
//
// subscribe: id1
// subscribe: id1
// subscribe: id2
// subscribe: id3
// publish: id1
// publish: id1
// publish: id2
// publish: id3
// publish to: id1
// publish to: id1
// publish to: id3
// unsubscribe: id1
// unsubscribe: id2
// unsubscribe: id3
```

## Options

### Concurrency
Concurrency indicates the number of the workers when Publish method is executed.
Default is 5.

### AllowDuplicableID
If AllowDuplicableID is true and tries to set the same ID Subscribers, those are subscribed as
slice to the same ID.
If not, override a old Subscriber to new Subscriber.
