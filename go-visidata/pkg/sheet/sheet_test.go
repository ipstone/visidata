package sheet

import "testing"

func TestInferColumnKinds(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"id", "price", "created_at", "label"})
	sh.AddRow([]string{"1", "12.50", "2026-03-30", "alpha"})
	sh.AddRow([]string{"2", "99.00", "2026-03-31", "beta"})
	sh.InferColumnKinds()

	want := []ValueKind{KindInt, KindFloat, KindDate, KindString}
	for i, col := range sh.Columns {
		if col.Kind != want[i] {
			t.Fatalf("column %d kind = %s, want %s", i, col.Kind, want[i])
		}
	}
}

func TestSummary(t *testing.T) {
	sh := New("data.csv", "/tmp/data.csv", []string{"a", "b"})
	sh.AddRow([]string{"1", "2"})
	if got, want := sh.Summary(), "data.csv: 1 row(s) x 2 column(s)"; got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}

func TestLooksLikeDate(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "2026-03-30", want: true},
		{value: "2026-03-30 17:59:44", want: true},
		{value: "7/3/2018 1:47p", want: true},
		{value: "7/3/2018 1:47pm", want: true},
		{value: "7/3/2018 1:47PM", want: true},
		{value: "not-a-date", want: false},
	}

	for _, tc := range tests {
		if got := looksLikeDate(tc.value); got != tc.want {
			t.Fatalf("looksLikeDate(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
