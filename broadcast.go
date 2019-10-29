package broadcast

import (
	"sync"

	"github.com/pkg/errors"
)

type ID interface{}

// Subscriber is subscribed to a publishment trigger in a Hub which is fired
// when Publish/PublishTo method called.
type Subscriber interface {
	// GetID returns a value that is gonna be used as the key of the map used to map ID and Subscriber.
	GetID() ID
	// Subscribe is called once when Subscriber subscribed.
	Subscribe() error
	// Unsubscribe is called once when Subscriber unsubscribed.
	Unsubscribe() error
}

// Hub holds Subscribers and publish functions to that.
type Hub struct {
	concurrency       int
	allowDuplicableID bool

	subs map[ID][]Subscriber
	mux  sync.Mutex
}

func (h *Hub) setSubs(id ID, subs []Subscriber) {
	h.mux.Lock()
	h.subs[id] = subs
	h.mux.Unlock()
}

func (h *Hub) clearSubs() {
	h.mux.Lock()
	h.subs = map[ID][]Subscriber{}
	h.mux.Unlock()
}

// New is the constructor of Hub.
func New(options ...Option) *Hub {
	h := &Hub{
		subs:              make(map[ID][]Subscriber),
		concurrency:       5,
		allowDuplicableID: true,
	}
	for _, o := range options {
		o(h)
	}
	return h
}

// Subscribe subscribes a Subscriber to a publishment trigger in a Hub which is fired
// when Publish/PublishTo method called
// If AllowDuplicableID option is false and exists a ID in an Hub already,
// override that as new Subscriber.
func (h *Hub) Subscribe(s Subscriber) error {
	id := s.GetID()
	old := h.subs[id]
	if !h.allowDuplicableID {
		for _, ss := range old {
			err := ss.Unsubscribe()
			if err != nil {
				return errors.Wrap(err, "unsubscribe error")
			}
		}
		h.setSubs(id, []Subscriber{s})
	} else {
		if err := s.Subscribe(); err != nil {
			return err
		}
		h.setSubs(id, append(h.subs[id], s))
	}
	return nil
}

// Unsubscribe unsubscribes Subscriber from a Hub.
func (h *Hub) Unsubscribe(s Subscriber) error {
	subs := h.subs[s.GetID()]
	for i, ss := range subs {
		if ss == s {
			if err := s.Unsubscribe(); err != nil {
				return err
			}
			h.subs[s.GetID()] = append(subs[:i], subs[i+1:]...)
		}
	}
	return nil
}

// Publish publishes to all Subscribers concurrently.
// Concurrency option of Hub can be used to set concurrency default is 5.
func (h *Hub) Publish(f func(Subscriber) error) error {
	var (
		ch     = make(chan Subscriber, len(h.subs))
		doneCh = make(chan struct{}, len(h.subs))
		errCh  = make(chan error)
	)

	var count int
	for _, s := range h.subs {
		for _, ss := range s {
			ch <- ss
		}
		count += len(s)
	}

	if count == 0 {
		close(ch)
		close(doneCh)
		close(errCh)
		return nil
	}

	for i := 0; i < h.concurrency; i++ {
		go func() {
			for {
				select {
				case s, ok := <-ch:
					if !ok {
						return
					}
					if err := f(s); err != nil {
						errCh <- err
						return
					}
					doneCh <- struct{}{}
				}
			}
		}()
	}

	var done int
	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			if done == count-1 {
				close(ch)
				close(doneCh)
				close(errCh)
				return nil
			}
			done++
		}
	}

}

// PublishTo publishes to specified IDs Subscribers.
func (h *Hub) PublishTo(f func(Subscriber) error, ids ...ID) error {
	for _, id := range ids {
		for _, s := range h.subs[id] {
			err := f(s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Terminate unsubscribes all Subscriber from a Hub.
func (h *Hub) Terminate() error {
	for _, s := range h.subs {
		for _, ss := range s {
			if err := ss.Unsubscribe(); err != nil {
				return err
			}
		}
	}
	h.clearSubs()
	return nil
}
