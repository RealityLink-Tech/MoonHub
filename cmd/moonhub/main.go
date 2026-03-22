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

	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/agent"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/auth"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/cron"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/gateway"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/migrate"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/model"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/onboard"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/skills"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/status"
	"github.com/RealityLink-Tech/MoonHub/cmd/moonhub/internal/version"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
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
