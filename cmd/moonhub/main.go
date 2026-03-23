// MoonHub - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 MoonHub contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/moonhub/cmd/moonhub/internal"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/agent"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/auth"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/cron"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/gateway"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/migrate"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/model"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/onboard"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/skills"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/status"
	"github.com/sipeed/moonhub/cmd/moonhub/internal/version"
	"github.com/sipeed/moonhub/pkg/config"
)

func NewMoonHubCommand() *cobra.Command {
	short := fmt.Sprintf("%s moonhub - Personal AI Assistant v%s\n\n", internal.Logo, config.GetVersion())

	cmd := &cobra.Command{
		Use:     "moonhub",
		Short:   short,
		Example: "moonhub version",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		model.NewModelCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorBlue = "\033[1;38;2;62;93;185m"
	colorRed  = "\033[1;38;2;213;70;70m"
	banner    = "\r\n" +
		colorBlue + "██████╗ ██╗ ██████╗ ██████╗ " + colorRed + " ██████╗██╗      █████╗ ██╗    ██╗\n" +
		colorBlue + "██╔══██╗██║██╔════╝██╔═══██╗" + colorRed + "██╔════╝██║     ██╔══██╗██║    ██║\n" +
		colorBlue + "██████╔╝██║██║     ██║   ██║" + colorRed + "██║     ██║     ███████║██║ █╗ ██║\n" +
		colorBlue + "██╔═══╝ ██║██║     ██║   ██║" + colorRed + "██║     ██║     ██╔══██║██║███╗██║\n" +
		colorBlue + "██║     ██║╚██████╗╚██████╔╝" + colorRed + "╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		colorBlue + "╚═╝     ╚═╝ ╚═════╝ ╚═════╝ " + colorRed + " ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n " +
		"\033[0m\r\n"
)

func main() {
	fmt.Printf("%s", banner)
	cmd := NewMoonHubCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
