package ansi

import "testing"

func TestIndexFields_ASCII(t *testing.T) {
	df := &DisplayFile{
		IsANSI: false,
		Data:   []byte("Hello {{TE1,8}} world\r\n"),
	}

	fields := IndexFields(df, 80)
	f, ok := fields["TE1"]
	if !ok {
		t.Fatalf("expected field TE1 to be found")
	}
	if f.Row != 1 || f.Col != 7 || f.MaxLen != 8 {
		t.Fatalf("unexpected field: %+v (want Row=1 Col=7 MaxLen=8)", f)
	}
}

func TestIndexFields_ANSI_CursorPosition(t *testing.T) {
	df := &DisplayFile{
		IsANSI: true,
		Data:   []byte("\x1b[10;20H{{TE1,8}}"),
	}

	fields := IndexFields(df, 80)
	f, ok := fields["TE1"]
	if !ok {
		t.Fatalf("expected field TE1 to be found")
	}
	if f.Row != 10 || f.Col != 20 || f.MaxLen != 8 {
		t.Fatalf("unexpected field: %+v (want Row=10 Col=20 MaxLen=8)", f)
	}
}

func TestBlankPlaceholders(t *testing.T) {
	in := []byte("A{{USER,30}}B")
	out := BlankPlaceholders(in)
	if len(out) != len(in) {
		t.Fatalf("length mismatch: got %d want %d", len(out), len(in))
	}
	if out[0] != 'A' || out[len(out)-1] != 'B' {
		t.Fatalf("unexpected sentinels: %q", string(out))
	}
	for i := 1; i < len(out)-1; i++ {
		if out[i] != ' ' {
			t.Fatalf("expected spaces in placeholder range, got %q at %d in %q", out[i], i, string(out))
		}
	}
	// Ensure input isn't modified
	if string(in) != "A{{USER,30}}B" {
		t.Fatalf("input was modified: %q", string(in))
	}
}
