package handlers

import (
	"errors"

	"context"

	"github.com/PastureStack/compose-cli/internal/events"
	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/logging"
	"github.com/PastureStack/compose-cli/project/options"
	"github.com/sirupsen/logrus"
)

func CreateStack(event *events.Event, apiClient *client.RancherClient) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": logging.SafeLogValue(event.ResourceID),
		"eventId":    logging.SafeLogValue(event.ID),
	})

	logger.Info("Stack Create Event Received")

	if err := createStack(logger, event, apiClient); err != nil {
		logger.Errorf("Stack Create Event Failed: %s", logging.SafeLogValue(err))
		publishTransitioningReply(err.Error(), event, apiClient, true)
		return err
	}

	logger.Info("Stack Create Event Done")
	return nil
}

func createStack(logger *logrus.Entry, event *events.Event, apiClient *client.RancherClient) error {
	stack, err := apiClient.Stack.ById(event.ResourceID)
	if err != nil {
		return err
	}

	if stack == nil {
		return errors.New("Failed to find stack")
	}

	if stack.DockerCompose == "" {
		return emptyReply(event, apiClient)
	}

	_, project, err := constructProject(logger, stack, apiClient.GetOpts().Url, apiClient.GetOpts().AccessKey, apiClient.GetOpts().SecretKey)
	if err != nil {
		return err
	}

	publishTransitioningReply("Creating stack", event, apiClient, false)

	if err := project.Create(context.Background(), options.Create{}); err != nil {
		return err
	}

	startOnCreate := false
	if fields, ok := stack.Data["fields"].(map[string]interface{}); ok {
		if on, ok := fields["startOnCreate"].(bool); ok {
			startOnCreate = on
		}
	}

	if startOnCreate {
		if err := project.Create(context.Background(), options.Create{}); err != nil {
			return err
		}

		if err := project.Up(context.Background(), options.Up{}); err != nil {
			return err
		}
	} else {
		// This is to make sure circular links work
		if err := project.Create(context.Background(), options.Create{}); err != nil {
			return err
		}
	}

	return emptyReply(event, apiClient)
}
