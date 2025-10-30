package model

import "testing"

func TestNewMessage_Validation(t *testing.T) {
	long := make([]byte, 141)
	for i := range long { long[i] = 'a' }
	if _, err := NewMessage("+905551112233", string(long)); err == nil {
		t.Fatal("expected error for >140 chars")
	}
	if _, err := NewMessage("+905551112233", "ok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
