package executor

import (
	"os"

	"github.com/PastureStack/compose-cli/executor/handlers"
	"github.com/PastureStack/compose-cli/internal/events"
	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/version"
	"github.com/sirupsen/logrus"
)

func Main() {
	logger := logrus.WithFields(logrus.Fields{
		"version": version.VERSION,
	})

	locale := environmentValue("PASTURESTACK_LOCALE", "")
	if locale == "" {
		locale = "en-US"
	}
	if locale != "en-US" && locale != "zh-TW" {
		logrus.Fatalf("unsupported locale %q; use en-US or zh-TW", locale)
	}
	logger.Info(executorMessage(locale, "start"))

	eventHandlers := map[string]events.EventHandler{
		"stack.create":        handlers.WithTimeout(handlers.CreateStack),
		"stack.upgrade":       handlers.WithTimeout(handlers.UpgradeStack),
		"stack.finishupgrade": handlers.WithTimeout(handlers.FinishUpgradeStack),
		"stack.rollback":      handlers.WithTimeout(handlers.RollbackStack),
		"ping": func(event *events.Event, apiClient *client.RancherClient) error {
			return nil
		},
	}

	router, err := events.NewEventRouter("compose-executor", 2000,
		environmentValue("PLATFORM_URL", "CATTLE_URL"),
		environmentValue("PLATFORM_ACCESS_KEY", "CATTLE_ACCESS_KEY"),
		environmentValue("PLATFORM_SECRET_KEY", "CATTLE_SECRET_KEY"),
		nil, eventHandlers, "stack", 250, events.DefaultPingConfig)
	if err != nil {
		logrus.WithField("error", err).Fatal("Unable to create event router")
	}

	if err := router.RemoveExternalHandlers("rancher-compose-executor"); err != nil {
		logrus.WithField("error", err).Fatal("Unable to remove previous event handler")
	}

	if err := router.Start(nil); err != nil {
		logrus.WithField("error", err).Fatal("Unable to start event router")
	}

	logger.Info(executorMessage(locale, "exit"))
}

func environmentValue(preferred, legacy string) string {
	if value := os.Getenv(preferred); value != "" {
		return value
	}
	if legacy != "" {
		return os.Getenv(legacy)
	}
	return ""
}

func executorMessage(locale, key string) string {
	messages := map[string]map[string]string{
		"en-US": {"start": "Starting Compose executor", "exit": "Compose executor stopped"},
		"zh-TW": {"start": "正在啟動 Compose 執行器", "exit": "Compose 執行器已停止"},
	}
	return messages[locale][key]
}
