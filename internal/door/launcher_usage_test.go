package door

import "testing"

func TestLauncherReserveDoor_SingleUserDeniedWhenInUse(t *testing.T) {
	l := NewLauncher("/usr/bin/dosemu", "./doors/drive_c", "./data/doors_tmp")

	cfg := &Config{Name: "Legend of Red Dragon", MultiUser: false}

	release1, err := l.reserveDoor(cfg)
	if err != nil {
		t.Fatalf("expected first reserve to succeed, got: %v", err)
	}
	defer release1()

	if got := l.UsersInDoor(cfg.Name); got != 1 {
		t.Fatalf("expected UsersInDoor=1, got %d", got)
	}

	_, err = l.reserveDoor(cfg)
	if err == nil {
		t.Fatalf("expected second reserve to fail for single-user door")
	}
}

func TestLauncherReserveDoor_MultiUserAllowsConcurrent(t *testing.T) {
	l := NewLauncher("/usr/bin/dosemu", "./doors/drive_c", "./data/doors_tmp")

	cfg := &Config{Name: "TradeWars", MultiUser: true}

	release1, err := l.reserveDoor(cfg)
	if err != nil {
		t.Fatalf("expected reserve1 to succeed, got: %v", err)
	}
	defer release1()

	release2, err := l.reserveDoor(cfg)
	if err != nil {
		t.Fatalf("expected reserve2 to succeed for multiuser door, got: %v", err)
	}
	defer release2()

	if got := l.UsersInDoor(cfg.Name); got != 2 {
		t.Fatalf("expected UsersInDoor=2, got %d", got)
	}
}

func TestLauncherReserveDoor_NormalizesDoorName(t *testing.T) {
	l := NewLauncher("/usr/bin/dosemu", "./doors/drive_c", "./data/doors_tmp")

	cfg := &Config{Name: "  DARKNESS ", MultiUser: false}
	release1, err := l.reserveDoor(cfg)
	if err != nil {
		t.Fatalf("expected reserve to succeed, got: %v", err)
	}
	defer release1()

	if got := l.UsersInDoor("darkness"); got != 1 {
		t.Fatalf("expected UsersInDoor('darkness')=1, got %d", got)
	}
	if got := l.UsersInDoor("DARKNESS"); got != 1 {
		t.Fatalf("expected UsersInDoor('DARKNESS')=1, got %d", got)
	}
}
