package allocation

import (
	"testing"
	"time"
)

func TestReplyAfterDeadlineIsNotEligible(t *testing.T) {
	deadline := time.Unix(10, 0)
	if replyWithinDeadline(deadline.Add(time.Nanosecond), deadline) {
		t.Fatal("late reply must not enter the auction")
	}
	if !replyWithinDeadline(deadline.Add(-time.Nanosecond), deadline) {
		t.Fatal("reply completed before deadline should remain eligible")
	}
}
