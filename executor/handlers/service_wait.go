package handlers

import (
	"fmt"
	"time"

	"github.com/PastureStack/compose-cli/internal/events"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

const (
	serviceWaitTimeout      = 5 * time.Minute
	servicePollInterval     = 500 * time.Millisecond
	serviceProgressInterval = 10 * time.Second
)

// Only observe the child after dispatch. Replaying the Stack handler while a
// child is transitioning could repeat a native mutation. Progress replies keep
// the original Engine event alive; its terminal reply stays with the handler.
type serviceWaiter struct {
	reload   func(*client.Service) error
	progress func() error
	now      func() time.Time
	sleep    func(time.Duration)
}

func newServiceWaiter(event *events.Event, apiClient *client.RancherClient) serviceWaiter {
	return serviceWaiter{
		reload: func(service *client.Service) error {
			return apiClient.Reload(&service.Resource, service)
		},
		progress: func() error {
			reply := newReply(event)
			reply.Transitioning = "yes"
			reply.TransitioningMessage = "Waiting for Stack services"
			return publishReply(reply, apiClient)
		},
		now:   time.Now,
		sleep: time.Sleep,
	}
}

func (w serviceWaiter) wait(service *client.Service) error {
	deadline := w.now().Add(serviceWaitTimeout)
	nextProgress := w.now()
	for {
		if err := w.reload(service); err != nil {
			return err
		}
		switch service.Transitioning {
		case "no":
			return nil
		case "yes":
			// Continue observing below without repeating an action.
		default:
			return fmt.Errorf("Waiting for %s failed: %s", service.Name, service.TransitioningMessage)
		}

		now := w.now()
		if !now.Before(deadline) {
			return fmt.Errorf("%w: %s", ErrTimeout, service.Name)
		}
		if !now.Before(nextProgress) {
			if err := w.progress(); err != nil {
				return err
			}
			nextProgress = now.Add(serviceProgressInterval)
		}
		delay := servicePollInterval
		if remaining := deadline.Sub(now); remaining < delay {
			delay = remaining
		}
		w.sleep(delay)
	}
}
