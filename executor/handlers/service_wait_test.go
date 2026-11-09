package handlers

import (
	"errors"
	"testing"
	"time"

	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func TestServiceWaiterUsesBoundedObservationAndKeepalive(t *testing.T) {
	for _, pendingReads := range []int{0, 8, 100} {
		t.Run(time.Duration(pendingReads).String(), func(t *testing.T) {
			now := time.Unix(0, 0)
			reads, progress, pauses := 0, 0, 0
			waiter := serviceWaiter{
				now: func() time.Time { return now },
				sleep: func(delay time.Duration) {
					if delay != servicePollInterval {
						t.Fatalf("unexpected observation interval: %s", delay)
					}
					now = now.Add(delay)
					pauses++
				},
				reload: func(s *client.Service) error {
					reads++
					s.Transitioning = "yes"
					if reads > pendingReads {
						s.Transitioning = "no"
					}
					return nil
				},
				progress: func() error { progress++; return nil },
			}
			if err := waiter.wait(&client.Service{}); err != nil {
				t.Fatal(err)
			}
			wantProgress := 0
			if pendingReads > 0 {
				wantProgress = (pendingReads-1)/20 + 1
			}
			if reads != pendingReads+1 || pauses != pendingReads || progress != wantProgress {
				t.Fatalf("reads/pauses/progress = %d/%d/%d", reads, pauses, progress)
			}
		})
	}
}

func TestServiceWaiterReportsDeadlineAndErrors(t *testing.T) {
	for _, failure := range []string{"deadline", "reload", "progress", "child"} {
		t.Run(failure, func(t *testing.T) {
			now := time.Unix(0, 0)
			reads, progress, pauses := 0, 0, 0
			original := errors.New("original failure")
			waiter := serviceWaiter{
				now:   func() time.Time { return now },
				sleep: func(delay time.Duration) { now = now.Add(delay); pauses++ },
				reload: func(s *client.Service) error {
					reads++
					s.Transitioning = "yes"
					if failure == "reload" {
						return original
					}
					if failure == "child" {
						s.Transitioning = "error"
						s.TransitioningMessage = "child failed"
					}
					return nil
				},
				progress: func() error {
					progress++
					if failure == "progress" {
						return original
					}
					return nil
				},
			}
			err := waiter.wait(&client.Service{Name: "one-child"})
			if err == nil {
				t.Fatal("failure was swallowed")
			}
			if failure == "deadline" {
				if !errors.Is(err, ErrTimeout) || now.Sub(time.Unix(0, 0)) != serviceWaitTimeout || reads != 601 || pauses != 600 || progress != 30 {
					t.Fatalf("unbounded or early timeout: %v, reads=%d pauses=%d progress=%d", err, reads, pauses, progress)
				}
			} else if failure == "reload" || failure == "progress" {
				if err != original {
					t.Fatalf("original failure replaced: %v", err)
				}
			}
			if failure != "deadline" && pauses != 0 {
				t.Fatal("retried a failed operation")
			}
		})
	}
}
