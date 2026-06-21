package postgres

import "testing"

func TestNullableIDKeepsLargeTelegramID(t *testing.T) {
	const telegramID int64 = 8640340960

	got := nullableID(telegramID)
	if !got.Valid {
		t.Fatal("nullableID returned invalid value for non-zero ID")
	}
	if got.Int64 != telegramID {
		t.Fatalf("nullableID Int64 = %d, want %d", got.Int64, telegramID)
	}
}

func TestNullableIDTreatsZeroAsNull(t *testing.T) {
	got := nullableID(0)
	if got.Valid {
		t.Fatalf("nullableID(0) Valid = true, want false")
	}
}
