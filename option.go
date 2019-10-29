package broadcast

// Option sets parameters to Hub.
type Option func(*Hub)

// Concurrency can be used to manage concurrency of the workers while doing Publish method.
// Default is 5.
func Concurrency(n int) Option {
	return func(h *Hub) {
		h.concurrency = n
	}
}

// AllowDuplicableID can be used to set if duplicable id into Hub's map.
// If AllowDuplicableID is true and tries to set the same ID Subscribers, those are subscribed as
// slice to the same ID.
// If not, override a old Subscriber to new Subscriber.
func AllowDuplicableID(b bool) Option {
	return func(h *Hub) {
		h.allowDuplicableID = b
	}
}
