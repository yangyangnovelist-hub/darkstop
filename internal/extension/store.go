package extension

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
)

// Order statuses as exposed in results and state. Cancelled orders are
// dropped from the store; StatusCancelled only ever appears in the
// CANCEL_ORDER action result.
const (
	StatusOpen      = "open"
	StatusExecuted  = "executed"
	StatusCancelled = "cancelled"

	// A trailing order may only activate from a recently observed, on-chain
	// validated FTSO sample. This is deliberately tighter than the vault's
	// settlement staleness ceiling because activation establishes its high.
	trailingSampleMaxAge = 10 * time.Second
)

// Order is a confidential stop-loss order held ONLY in enclave memory.
// TriggerPrice (USD per FLR, 6 decimals) never leaves the TEE until
// settlement reveals it on-chain.
type Order struct {
	ID            uint64
	TriggerPrice  *big.Int
	TrailBps      uint16
	HighWatermark *big.Int
	Status        string
}

// Store is an in-memory order book for the simulated-TEE mode. Restart
// recovery is documented roadmap, deliberately not built.
type Store struct {
	mu          sync.RWMutex
	orders      map[uint64]Order
	latestPrice *big.Int
	latestAt    time.Time
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
	if o.ID == 0 || o.Status != StatusOpen {
		return fmt.Errorf("invalid order identity or status")
	}
	isFixed := o.TriggerPrice != nil && o.TriggerPrice.Sign() > 0 && o.TriggerPrice.BitLen() <= 256 && o.TrailBps == 0 && o.HighWatermark == nil
	isTrailing := o.TriggerPrice == nil && o.TrailBps >= 25 && o.TrailBps <= 5000 && o.HighWatermark == nil && s.trailingReadyLocked(time.Now())
	if !isFixed && !isTrailing {
		return fmt.Errorf("order must contain one valid fixed or trailing policy")
	}
	if isTrailing && s.latestPrice != nil {
		o.HighWatermark = new(big.Int).Set(s.latestPrice)
	}
	s.orders[o.ID] = cloneOrder(o)
	return nil
}

// Get returns the order with the given id, if present.
func (s *Store) Get(id uint64) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return cloneOrder(o), ok
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
			open = append(open, cloneOrder(o))
		}
	}
	sort.Slice(open, func(i, j int) bool { return open[i].ID < open[j].ID })
	return open
}

// ObservePrice records the watcher's latest trusted FTSO sample. New trailing
// orders activate from this value instead of waiting for a later polling tick.
func (s *Store) ObservePrice(price *big.Int, observedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if price != nil && price.Sign() > 0 && !observedAt.IsZero() {
		s.latestPrice = new(big.Int).Set(price)
		s.latestAt = observedAt
	}
}

// SupportedPolicies is also a readiness signal: trailing is advertised only
// while a fresh trusted sample is available to seed a new order safely.
func (s *Store) SupportedPolicies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policies := []string{"fixed"}
	if s.trailingReadyLocked(time.Now()) {
		policies = append(policies, "trailing")
	}
	return policies
}

func (s *Store) trailingReadyLocked(now time.Time) bool {
	if s.latestPrice == nil || s.latestAt.IsZero() {
		return false
	}
	age := now.Sub(s.latestAt)
	return age >= 0 && age <= trailingSampleMaxAge
}

// TriggeredOrders updates private trailing-stop high-watermarks and returns
// the orders whose effective trigger has been crossed. The effective trigger
// is copied into TriggerPrice for on-chain FTSO verification at settlement.
func (s *Store) TriggeredOrders(price *big.Int) []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	if price == nil || price.Sign() <= 0 {
		return nil
	}
	triggered := make([]Order, 0)
	for id, o := range s.orders {
		if o.Status != StatusOpen {
			continue
		}
		effective := o.TriggerPrice
		if o.TrailBps > 0 {
			if o.HighWatermark == nil || price.Cmp(o.HighWatermark) > 0 {
				o.HighWatermark = new(big.Int).Set(price)
				s.orders[id] = o
			}
			effective = new(big.Int).Mul(o.HighWatermark, big.NewInt(int64(10_000-o.TrailBps)))
			effective.Div(effective, big.NewInt(10_000))
		}
		if effective != nil && price.Cmp(effective) <= 0 {
			o.TriggerPrice = new(big.Int).Set(effective)
			triggered = append(triggered, cloneOrder(o))
		}
	}
	sort.Slice(triggered, func(i, j int) bool { return triggered[i].ID < triggered[j].ID })
	return triggered
}

// EffectiveTrigger returns the current private trigger for an open order.
// For trailing orders it is derived from the highest trusted FTSO sample
// observed so far. The returned integer is always an independent copy.
func (s *Store) EffectiveTrigger(id uint64) (*big.Int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok || o.Status != StatusOpen {
		return nil, false
	}
	if o.TrailBps == 0 {
		if o.TriggerPrice == nil {
			return nil, false
		}
		return new(big.Int).Set(o.TriggerPrice), true
	}
	if o.HighWatermark == nil {
		return nil, false
	}
	effective := new(big.Int).Mul(o.HighWatermark, big.NewInt(int64(10_000-o.TrailBps)))
	effective.Div(effective, big.NewInt(10_000))
	return effective, true
}

// Snapshot returns all orders sorted by id.
func (s *Store) Snapshot() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Order, 0, len(s.orders))
	for _, o := range s.orders {
		all = append(all, cloneOrder(o))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}

func cloneOrder(o Order) Order {
	if o.TriggerPrice != nil {
		o.TriggerPrice = new(big.Int).Set(o.TriggerPrice)
	}
	if o.HighWatermark != nil {
		o.HighWatermark = new(big.Int).Set(o.HighWatermark)
	}
	return o
}
