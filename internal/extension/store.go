package extension

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
)

// Order statuses as exposed in results and state. Cancelled orders are
// dropped from the store; StatusCancelled only ever appears in the
// CANCEL_ORDER action result.
const (
	StatusOpen      = "open"
	StatusExecuted  = "executed"
	StatusCancelled = "cancelled"
)

// Order is a confidential stop-loss order held ONLY in enclave memory.
// TriggerPrice (USD per FLR, 6 decimals) never leaves the TEE until
// settlement reveals it on-chain.
type Order struct {
	ID           uint64
	TriggerPrice *big.Int
	Status       string
}

// Store is an in-memory order book for the simulated-TEE mode. Restart
// recovery is documented roadmap, deliberately not built.
type Store struct {
	mu     sync.RWMutex
	orders map[uint64]Order
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{orders: make(map[uint64]Order)}
}

// Put adds a new order; it fails if the id already exists.
func (s *Store) Put(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.orders[o.ID]; exists {
		return fmt.Errorf("order %d already exists", o.ID)
	}
	s.orders[o.ID] = o
	return nil
}

// Get returns the order with the given id, if present.
func (s *Store) Get(id uint64) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

// Delete removes an order (cancellation). Returns false if it was absent.
func (s *Store) Delete(id uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[id]; !ok {
		return false
	}
	delete(s.orders, id)
	return true
}

// MarkExecuted flips an open order to executed. Returns false if the order
// is absent or not open.
func (s *Store) MarkExecuted(id uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok || o.Status != StatusOpen {
		return false
	}
	o.Status = StatusExecuted
	s.orders[id] = o
	return true
}

// OpenOrders returns all open orders sorted by id.
func (s *Store) OpenOrders() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	open := make([]Order, 0, len(s.orders))
	for _, o := range s.orders {
		if o.Status == StatusOpen {
			open = append(open, o)
		}
	}
	sort.Slice(open, func(i, j int) bool { return open[i].ID < open[j].ID })
	return open
}

// Snapshot returns all orders sorted by id.
func (s *Store) Snapshot() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Order, 0, len(s.orders))
	for _, o := range s.orders {
		all = append(all, o)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}
