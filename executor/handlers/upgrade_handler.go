package handlers

import (
	"errors"

	"context"

	"github.com/PastureStack/compose-cli/internal/events"
	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/logging"
	"github.com/PastureStack/compose-cli/project/options"
	"github.com/PastureStack/compose-cli/utils"
	"github.com/sirupsen/logrus"
)

func UpgradeStack(event *events.Event, apiClient *client.RancherClient) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": logging.SafeLogValue(event.ResourceID),
		"eventId":    logging.SafeLogValue(event.ID),
	})

	logger.Info("Upgrade Stack Event Received")

	if err := upgradeEnvironment(logger, event, apiClient); err != nil {
		logger.Errorf("Stack Upgrade Event Failed: %s", logging.SafeLogValue(err))
		return err
	}

	logger.Info("Stack Upgrade Event Done")
	return nil
}

func FinishUpgradeStack(event *events.Event, apiClient *client.RancherClient) error {
	return finishUpgradeStack(event, apiClient, newServiceWaiter(event, apiClient).wait)
}

func finishUpgradeStack(event *events.Event, apiClient *client.RancherClient, wait func(*client.Service) error) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": logging.SafeLogValue(event.ResourceID),
		"eventId":    logging.SafeLogValue(event.ID),
	})

	logger.Info("Finish Upgrade Stack Event Received")

	stack, err := apiClient.Stack.ById(event.ResourceID)
	if err != nil {
		return err
	}

	if stack == nil {
		return errors.New("Failed to find stack")
	}

	var services client.ServiceCollection
	if err := apiClient.GetLink(stack.Resource, "services", &services); err != nil {
		return err
	}

	for _, service := range services.Data {
		if err := wait(&service); err != nil {
			return err
		}
		if service.State == "upgraded" {
			service, err := apiClient.Service.ActionFinishupgrade(&service)
			if err != nil {
				return err
			}
			if err := wait(service); err != nil {
				return err
			}
		}
	}

	logger.Info("Finish Stack Upgrade Event Done")
	return reply(event, apiClient, map[string]interface{}{
		"previousExternalId":  nil,
		"previousEnvironment": nil,
	})
}

func RollbackStack(event *events.Event, apiClient *client.RancherClient) error {
	return rollbackStack(event, apiClient, newServiceWaiter(event, apiClient).wait)
}

func rollbackStack(event *events.Event, apiClient *client.RancherClient, wait func(*client.Service) error) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": logging.SafeLogValue(event.ResourceID),
		"eventId":    logging.SafeLogValue(event.ID),
	})

	logger.Info("Rollback Stack Event Received")

	stack, err := apiClient.Stack.ById(event.ResourceID)
	if err != nil {
		return err
	}

	if stack == nil {
		return errors.New("Failed to find stack")
	}

	var services client.ServiceCollection
	if err := apiClient.GetLink(stack.Resource, "services", &services); err != nil {
		return err
	}

	for _, service := range services.Data {
		if err := wait(&service); err != nil {
			return err
		}
		if service.State == "upgraded" || service.State == "canceled-upgrade" {
			service, err := apiClient.Service.ActionRollback(&service)
			if err != nil {
				return err
			}
			if err := wait(service); err != nil {
				return err
			}
		}
	}

	logger.Info("Rollback Stack Event Done")
	newId := stack.PreviousExternalId
	if newId == "" {
		newId = stack.ExternalId
	}
	newEnv := stack.PreviousEnvironment
	if len(newEnv) == 0 {
		newEnv = stack.Environment
	}

	return reply(event, apiClient, map[string]interface{}{
		"previousExternalId":  nil,
		"previousEnvironment": nil,
		"externalId":          newId,
		"environment":         newEnv,
	})
}

func upgradeEnvironment(logger *logrus.Entry, event *events.Event, apiClient *client.RancherClient) error {
	var upgradeOpts client.StackUpgrade

	if err := utils.ConvertByJSON(event.Data, &upgradeOpts); err != nil {
		return err
	}

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

	project, newEnv, err := constructProjectUpgrade(logger, stack, upgradeOpts, apiClient.GetOpts().Url, apiClient.GetOpts().AccessKey, apiClient.GetOpts().SecretKey)
	if err != nil {
		return err
	}

	publishTransitioningReply("Upgrading stack", event, apiClient, false)

	if err := project.Up(context.Background(), options.Up{}); err != nil {
		return err
	}

	previous := stack.PreviousExternalId
	if previous == "" {
		previous = stack.ExternalId
	}

	previousEnv := stack.PreviousEnvironment
	if len(previousEnv) == 0 {
		previousEnv = stack.Environment
	}

	return reply(event, apiClient, map[string]interface{}{
		"externalId":          upgradeOpts.ExternalId,
		"environment":         newEnv,
		"previousExternalId":  previous,
		"previousEnvironment": previousEnv,
	})
}
