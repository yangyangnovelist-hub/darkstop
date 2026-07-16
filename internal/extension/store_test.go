package extension

import (
	"math/big"
	"sync"
	"testing"
)

func TestStore_PutAndGet(t *testing.T) {
	s := NewStore()

	if err := s.Put(Order{ID: 1, TriggerPrice: big.NewInt(20000), Status: StatusOpen}); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, ok := s.Get(1)
	if !ok {
		t.Fatal("expected order 1 to exist")
	}
	if got.TriggerPrice.Cmp(big.NewInt(20000)) != 0 {
		t.Errorf("trigger mismatch: %s", got.TriggerPrice)
	}
	if got.Status != StatusOpen {
		t.Errorf("expected status open, got %s", got.Status)
	}

	if _, ok := s.Get(2); ok {
		t.Error("expected order 2 to be absent")
	}
}

func TestStore_PutDuplicateFails(t *testing.T) {
	s := NewStore()

	if err := s.Put(Order{ID: 1, TriggerPrice: big.NewInt(1), Status: StatusOpen}); err != nil {
		t.Fatalf("first put: %v", err)
	}
	if err := s.Put(Order{ID: 1, TriggerPrice: big.NewInt(2), Status: StatusOpen}); err == nil {
		t.Error("expected duplicate put to fail")
	}
	// Original order untouched.
	got, _ := s.Get(1)
	if got.TriggerPrice.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("duplicate put overwrote order: trigger %s", got.TriggerPrice)
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()
	_ = s.Put(Order{ID: 1, TriggerPrice: big.NewInt(1), Status: StatusOpen})

	if !s.Delete(1) {
		t.Error("expected delete of existing order to return true")
	}
	if _, ok := s.Get(1); ok {
		t.Error("expected order to be gone after delete")
	}
	if s.Delete(1) {
		t.Error("expected delete of missing order to return false")
	}
}

func TestStore_MarkExecuted(t *testing.T) {
	s := NewStore()
	_ = s.Put(Order{ID: 1, TriggerPrice: big.NewInt(1), Status: StatusOpen})

	if !s.MarkExecuted(1) {
		t.Error("expected MarkExecuted on open order to return true")
	}
	got, _ := s.Get(1)
	if got.Status != StatusExecuted {
		t.Errorf("expected status executed, got %s", got.Status)
	}

	if s.MarkExecuted(1) {
		t.Error("expected MarkExecuted on already-executed order to return false")
	}
	if s.MarkExecuted(99) {
		t.Error("expected MarkExecuted on missing order to return false")
	}
}

func TestStore_OpenOrdersSortedAndFiltered(t *testing.T) {
	s := NewStore()
	_ = s.Put(Order{ID: 3, TriggerPrice: big.NewInt(3), Status: StatusOpen})
	_ = s.Put(Order{ID: 1, TriggerPrice: big.NewInt(1), Status: StatusOpen})
	_ = s.Put(Order{ID: 2, TriggerPrice: big.NewInt(2), Status: StatusOpen})
	s.MarkExecuted(2)

	open := s.OpenOrders()
	if len(open) != 2 {
		t.Fatalf("expected 2 open orders, got %d", len(open))
	}
	if open[0].ID != 1 || open[1].ID != 3 {
		t.Errorf("expected ids [1 3], got [%d %d]", open[0].ID, open[1].ID)
	}
}

func TestStore_SnapshotSorted(t *testing.T) {
	s := NewStore()
	_ = s.Put(Order{ID: 2, TriggerPrice: big.NewInt(2), Status: StatusOpen})
	_ = s.Put(Order{ID: 1, TriggerPrice: big.NewInt(1), Status: StatusOpen})
	s.MarkExecuted(1)

	all := s.Snapshot()
	if len(all) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(all))
	}
	if all[0].ID != 1 || all[0].Status != StatusExecuted {
		t.Errorf("unexpected first order: %+v", all[0])
	}
	if all[1].ID != 2 || all[1].Status != StatusOpen {
		t.Errorf("unexpected second order: %+v", all[1])
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := uint64(1); i <= 50; i++ {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			_ = s.Put(Order{ID: id, TriggerPrice: big.NewInt(int64(id)), Status: StatusOpen})
			s.OpenOrders()
			s.MarkExecuted(id)
			s.Snapshot()
		}(i)
	}
	wg.Wait()

	if got := len(s.Snapshot()); got != 50 {
		t.Errorf("expected 50 orders, got %d", got)
	}
}
