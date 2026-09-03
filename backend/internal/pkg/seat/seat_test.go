package seat

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		row int
		col string
		ok  bool
	}{
		{row: 1, col: "A", ok: true},
		{row: 12, col: "A", ok: true},
		{row: 42, col: "F", ok: true},
		{row: 3, col: "Z", ok: true},
	}
	for _, c := range cases {
		id := EncodeID(c.row, c.col)
		if id <= 0 {
			t.Fatalf("EncodeID(%d, %q) = %d, want > 0", c.row, c.col, id)
		}
		label := DecodeLabel(id)
		if label != itoa(c.row)+c.col {
			t.Fatalf("DecodeLabel(EncodeID(%d,%q)) = %q, want %q", c.row, c.col, label, itoa(c.row)+c.col)
		}
		row, col, ok := ParseLabel(label)
		if !ok || row != c.row || col != c.col {
			t.Fatalf("ParseLabel(%q) = %d,%q,%v, want %d,%q,true", label, row, col, ok, c.row, c.col)
		}
	}
}

func TestEncodeIDInvalid(t *testing.T) {
	if id := EncodeID(0, "A"); id != 0 {
		t.Fatalf("EncodeID(0, A) = %d, want 0", id)
	}
	if id := EncodeID(1, "AA"); id != 0 {
		t.Fatalf("EncodeID(1, AA) = %d, want 0", id)
	}
	if id := EncodeID(1, "a1"); id != 0 {
		t.Fatalf("EncodeID with invalid col = %d, want 0", id)
	}
}

func TestDecodeLabelInvalid(t *testing.T) {
	for _, id := range []int64{0, -5, 3, 20050} {
		if label := DecodeLabel(id); label != "" {
			t.Fatalf("DecodeLabel(%d) = %q, want empty", id, label)
		}
	}
}

func TestParseLabelInvalid(t *testing.T) {
	for _, label := range []string{"", "A", "12", "12AA", "0A", "12 1", "12!"} {
		if row, col, ok := ParseLabel(label); ok {
			t.Fatalf("ParseLabel(%q) unexpectedly ok: %d,%q", label, row, col)
		}
	}
}

func TestParseLabelLowercase(t *testing.T) {
	row, col, ok := ParseLabel("12a")
	if !ok || row != 12 || col != "A" {
		t.Fatalf("ParseLabel(12a) = %d,%q,%v", row, col, ok)
	}
}

func TestEncodeUniqueAcrossRows(t *testing.T) {
	seen := map[int64]bool{}
	for row := 1; row <= 60; row++ {
		for _, col := range []string{"A", "B", "C", "D", "E", "F"} {
			id := EncodeID(row, col)
			if seen[id] {
				t.Fatalf("duplicate id %d for %d%s", id, row, col)
			}
			seen[id] = true
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
