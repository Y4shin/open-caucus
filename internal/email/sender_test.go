package email

import (
	"context"
	"testing"
)

func TestMockSender_RecordsSentEmails(t *testing.T) {
	s := &MockSender{}

	if !s.Enabled() {
		t.Fatal("MockSender should report enabled")
	}
	if len(s.Sent) != 0 {
		t.Fatalf("expected 0 sent, got %d", len(s.Sent))
	}

	err := s.Send(context.Background(), "alice@example.com", "Test Subject", "<p>html</p>", "text")
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if len(s.Sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(s.Sent))
	}
	if s.Sent[0].To != "alice@example.com" {
		t.Errorf("expected to=alice@example.com, got %q", s.Sent[0].To)
	}
	if s.Sent[0].Subject != "Test Subject" {
		t.Errorf("expected subject='Test Subject', got %q", s.Sent[0].Subject)
	}
	if s.Sent[0].HTMLBody != "<p>html</p>" {
		t.Errorf("expected html body, got %q", s.Sent[0].HTMLBody)
	}
	if s.Sent[0].TextBody != "text" {
		t.Errorf("expected text body, got %q", s.Sent[0].TextBody)
	}
}

func TestMockSender_MultipleEmails(t *testing.T) {
	s := &MockSender{}
	for i := 0; i < 5; i++ {
		_ = s.Send(context.Background(), "user@example.com", "Subj", "", "")
	}
	if len(s.Sent) != 5 {
		t.Fatalf("expected 5 sent, got %d", len(s.Sent))
	}
}

func TestNoopSender_ReturnsError(t *testing.T) {
	s := &NoopSender{}

	if s.Enabled() {
		t.Fatal("NoopSender should report disabled")
	}

	err := s.Send(context.Background(), "x@y.z", "s", "h", "t")
	if err == nil {
		t.Fatal("NoopSender.Send should return error")
	}
}

func TestNewSender_NilConfig_ReturnsNoop(t *testing.T) {
	s := NewSender(nil)
	if s.Enabled() {
		t.Fatal("nil config should produce disabled sender")
	}
}
