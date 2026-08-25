package main

import (
	"fmt"
	"os"
	"path"

	composeApp "github.com/PastureStack/compose-cli/app"
	"github.com/PastureStack/compose-cli/executor"
	cli "github.com/PastureStack/compose-cli/internal/legacycli"
	"github.com/PastureStack/compose-cli/version"
	"github.com/sirupsen/logrus"
)

func beforeApp(c *cli.Context) error {
	locale := c.GlobalString("locale")
	if locale != "en-US" && locale != "zh-TW" {
		return fmt.Errorf("unsupported locale %q; use en-US or zh-TW", locale)
	}
	if c.GlobalBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Info(operatorMessage(locale, "ready"))
	return nil
}

func main() {
	base := path.Base(os.Args[0])
	if base == "compose-executor" || base == "rancher-compose-executor" {
		if requestsVersion(os.Args) {
			fmt.Printf("compose-executor version %s\n", version.VERSION)
			return
		}
		executor.Main()
	} else {
		cliMain()
	}
}

func requestsVersion(args []string) bool {
	return len(args) == 2 && (args[1] == "--version" || args[1] == "-v")
}

func cliMain() {
	factory := &composeApp.PlatformProjectFactory{}

	app := cli.NewApp()
	app.Name = "pasturestack-compose"
	app.Usage = "Deploy Docker Compose workloads through a compatible control-platform API"
	app.Version = version.VERSION
	app.Author = "PastureStack community"
	app.Email = ""
	app.Before = beforeApp
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "locale",
			Value:  "en-US",
			Usage:  "Operator message locale: en-US or zh-TW",
			EnvVar: "PASTURESTACK_LOCALE",
		},
		cli.BoolFlag{
			Name: "verbose,debug",
		},
		cli.StringSliceFlag{
			Name:   "file,f",
			Usage:  "Specify one or more alternate compose files (default: docker-compose.yml)",
			Value:  &cli.StringSlice{},
			EnvVar: "COMPOSE_FILE",
		},
		cli.StringFlag{
			Name:   "project-name,p",
			Usage:  "Specify an alternate project name (default: directory name)",
			EnvVar: "COMPOSE_PROJECT_NAME",
		},
		cli.StringFlag{
			Name: "url",
			Usage: fmt.Sprintf(
				"Specify the control-platform API endpoint URL",
			),
			EnvVar: "PLATFORM_URL,RANCHER_URL",
		},
		cli.StringFlag{
			Name: "access-key",
			Usage: fmt.Sprintf(
				"Specify the control-platform API access key",
			),
			EnvVar: "PLATFORM_ACCESS_KEY,RANCHER_ACCESS_KEY",
		},
		cli.StringFlag{
			Name: "secret-key",
			Usage: fmt.Sprintf(
				"Specify the control-platform API secret key",
			),
			EnvVar: "PLATFORM_SECRET_KEY,RANCHER_SECRET_KEY",
		},
		cli.StringFlag{
			Name:  "platform-file,rancher-file,r",
			Usage: "Specify an alternate platform compatibility file (default: platform-compose.yml)",
		},
		cli.StringFlag{
			Name:  "env-file,e",
			Usage: "Specify a file from which to read environment variables",
		},
		cli.StringFlag{
			Name:  "bindings-file,b",
			Usage: "Specify a file from which to read bindings",
		},
	}
	app.Commands = []cli.Command{
		composeApp.CreateCommand(factory),
		composeApp.UpCommand(factory),
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func operatorMessage(locale, key string) string {
	messages := map[string]map[string]string{
		"en-US": {"ready": "PastureStack Compose CLI is ready"},
		"zh-TW": {"ready": "PastureStack Compose 命令列工具已就緒"},
	}
	return messages[locale][key]
}
