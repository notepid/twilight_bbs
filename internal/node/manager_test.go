package node

import "testing"

func TestManagerAcquireLowestAvailable(t *testing.T) {
	mgr := NewManager(3, "TestBBS", "Sysop")

	id, ok := mgr.Acquire()
	if !ok || id != 1 {
		t.Fatalf("expected id=1 ok=true, got id=%d ok=%v", id, ok)
	}
	mgr.Add(&Node{ID: id})

	id, ok = mgr.Acquire()
	if !ok || id != 2 {
		t.Fatalf("expected id=2 ok=true, got id=%d ok=%v", id, ok)
	}
	mgr.Add(&Node{ID: id})

	mgr.Remove(1)

	id, ok = mgr.Acquire()
	if !ok || id != 1 {
		t.Fatalf("expected reused id=1 ok=true, got id=%d ok=%v", id, ok)
	}
}

func TestManagerAcquireCapacityAndReuse(t *testing.T) {
	mgr := NewManager(2, "TestBBS", "Sysop")

	id1, ok := mgr.Acquire()
	if !ok || id1 != 1 {
		t.Fatalf("expected id=1 ok=true, got id=%d ok=%v", id1, ok)
	}
	mgr.Add(&Node{ID: id1})

	id2, ok := mgr.Acquire()
	if !ok || id2 != 2 {
		t.Fatalf("expected id=2 ok=true, got id=%d ok=%v", id2, ok)
	}
	mgr.Add(&Node{ID: id2})

	id3, ok := mgr.Acquire()
	if ok || id3 != 0 {
		t.Fatalf("expected id=0 ok=false when full, got id=%d ok=%v", id3, ok)
	}

	mgr.Remove(id1)

	id4, ok := mgr.Acquire()
	if !ok || id4 != 1 {
		t.Fatalf("expected reused id=1 ok=true, got id=%d ok=%v", id4, ok)
	}
}
