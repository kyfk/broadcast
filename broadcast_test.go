package broadcast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("assert default values", func(t *testing.T) {
		hub := New()
		assert.Equal(t, 5, hub.concurrency)
		assert.Equal(t, make(map[ID][]Subscriber), hub.subs)
		assert.Equal(t, true, hub.allowDuplicableID)
	})

	t.Run("assert constructed values", func(t *testing.T) {
		hub := New(
			Concurrency(100),
			AllowDuplicableID(false),
		)
		assert.Equal(t, 100, hub.concurrency)
		assert.Equal(t, false, hub.allowDuplicableID)
	})
}

func TestSetSubs(t *testing.T) {
	hub := New()
	assert.NotPanics(t, func() {
		for i := 0; i < 100; i++ {
			go func() {
				hub.setSubs("some id", []Subscriber{})
			}()
		}
	})
}

func TestClearSubs(t *testing.T) {
	hub := New()
	assert.NotPanics(t, func() {
		for i := 0; i < 300; i++ {
			go func() {
				hub.clearSubs()
			}()
		}
	})
}

func TestSubscribe(t *testing.T) {
	t.Run("if duplicable id is not allowed and is same id", func(t *testing.T) {
		hub := New(AllowDuplicableID(false))
		t.Run("override old one by new one", func(t *testing.T) {
			subscriber1 := &stabSubscriber{id: "id1"}
			subscriber2 := &stabSubscriber{id: "id1"}
			assert.NoError(t, hub.Subscribe(subscriber1))
			assert.Equal(t, []Subscriber{subscriber1}, hub.subs["id1"])
			assert.NoError(t, hub.Subscribe(subscriber2))
			assert.Equal(t, []Subscriber{subscriber2}, hub.subs["id1"])
		})
		t.Run("if the Unsubscribe method returns an error, returns an error", func(t *testing.T) {
			subscriber1 := &stabSubscriber{id: "id1", unsubscribeErr: errors.New("error")}
			subscriber2 := &stabSubscriber{id: "id1"}
			assert.NoError(t, hub.Subscribe(subscriber1))
			assert.Equal(t, []Subscriber{subscriber1}, hub.subs["id1"])
			assert.Error(t, hub.Subscribe(subscriber2))
			assert.Equal(t, []Subscriber{subscriber1}, hub.subs["id1"])
		})
	})

	t.Run("if duplicable id is allowed", func(t *testing.T) {
		hub := New(AllowDuplicableID(true))
		t.Run("can append subscribers as same id", func(t *testing.T) {
			subscriber1 := &stabSubscriber{id: "id1"}
			subscriber2 := &stabSubscriber{id: "id1"}
			assert.NoError(t, hub.Subscribe(subscriber1))
			assert.Equal(t, []Subscriber{subscriber1}, hub.subs["id1"])
			assert.NoError(t, hub.Subscribe(subscriber2))
			assert.Equal(t, []Subscriber{subscriber1, subscriber2}, hub.subs["id1"])
		})
		t.Run("if the Subscribe method returns an error, return an error", func(t *testing.T) {
			subscriber1 := &stabSubscriber{id: "id1", subscribeErr: errors.New("error")}
			assert.Error(t, hub.Subscribe(subscriber1))
		})
	})
}

func TestUnsubscribe(t *testing.T) {
	hub := New()
	subscriber1 := &stabSubscriber{id: "id1"}
	subscriber2 := &stabSubscriber{id: "id1"}
	subscriber3 := &stabSubscriber{id: "id1"}
	assert.NoError(t, hub.Subscribe(subscriber1))
	assert.NoError(t, hub.Subscribe(subscriber2))
	assert.Equal(t, []Subscriber{subscriber1, subscriber2}, hub.subs["id1"])

	assert.NoError(t, hub.Unsubscribe(subscriber3))
	assert.Equal(t, []Subscriber{subscriber1, subscriber2}, hub.subs["id1"])

	assert.NoError(t, hub.Unsubscribe(subscriber2))
	assert.Equal(t, []Subscriber{subscriber1}, hub.subs["id1"])

	t.Run("if the Unsubscribe method of Subscriber returns an error, returns an error", func(t *testing.T) {
		hub := New()
		subscriberErr := &stabSubscriber{id: "id1", unsubscribeErr: errors.New("error")}
		assert.NoError(t, hub.Subscribe(subscriberErr))
		assert.Error(t, hub.Unsubscribe(subscriberErr))
	})
}

func ExampleHub_Publish() {
	hub := New()
	hub.Subscribe(&stabSubscriber{id: "id1"})
	hub.Subscribe(&stabSubscriber{id: "id2"})
	hub.Subscribe(&stabSubscriber{id: "id3"})

	hub.Publish(func(s Subscriber) error {
		fmt.Println(s.(*stabSubscriber).id)
		return nil
	})
	// Unordered output: id1
	// id2
	// id3
}

func TestPublish(t *testing.T) {
	hub := New()
	hub.Subscribe(&stabSubscriber{id: "id1"})
	err := hub.Publish(func(s Subscriber) error {
		return errors.New("error")
	})
	assert.Error(t, err)
}

func TestPublishTo(t *testing.T) {
	hub := New()
	hub.Subscribe(&stabSubscriber{id: "id1"})
	hub.Subscribe(&stabSubscriber{id: "id2"})
	hub.Subscribe(&stabSubscriber{id: "id3"})
	assert.NoError(t, hub.PublishTo(func(s Subscriber) error {
		switch s.(*stabSubscriber).id {
		case "id1", "id2":
			return nil
		default:
			return errors.New("error")
		}
	}, "id1", "id2"))
	assert.Error(t, hub.PublishTo(func(s Subscriber) error {
		return errors.New("error")
	}, "id1", "id2"))
}

func ExampleHub_Terminate() {
	hub := New()
	hub.Subscribe(&stabSubscriber{id: "id1"})
	hub.Subscribe(&stabSubscriber{id: "id2"})
	hub.Subscribe(&stabSubscriber{id: "id3"})

	hub.Terminate()

	// Output:
	//
	// unsubscribe: id1
	// unsubscribe: id2
	// unsubscribe: id3
}

func TestTerminate(t *testing.T) {
	hub := New()
	hub.Subscribe(&stabSubscriber{id: "id1", unsubscribeErr: errors.New("error")})

	assert.Error(t, hub.Terminate())
}

type stabSubscriber struct {
	id             string
	subscribeErr   error
	unsubscribeErr error
}

func (s stabSubscriber) GetID() ID { return s.id }
func (s stabSubscriber) Subscribe() error {
	return s.subscribeErr
}
func (s stabSubscriber) Unsubscribe() error {
	fmt.Println("unsubscribe:", s.id)
	return s.unsubscribeErr
}
