package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/github"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var projectInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a max-ops.yaml in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		board, _ := cmd.Flags().GetInt("project")
		owner, _ := cmd.Flags().GetString("owner")
		hubber, _ := cmd.Flags().GetString("hubber")
		opsChannel, _ := cmd.Flags().GetString("ops-channel")
		sotPath, _ := cmd.Flags().GetString("sot")

		noFlags := board == 0 && owner == "" && hubber == "" && opsChannel == "" && sotPath == ""

		if noFlags {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Project board number: ")
			boardText, _ := reader.ReadString('\n')
			boardText = strings.TrimSpace(boardText)
			if boardText != "" {
				parsed, err := strconv.Atoi(boardText)
				if err != nil {
					return fmt.Errorf("invalid project board: %w", err)
				}
				board = parsed
			}

			fmt.Print("Project owner: ")
			ownerText, _ := reader.ReadString('\n')
			owner = strings.TrimSpace(ownerText)

			defaultHubber := ""
			if hubber == "" {
				if user, err := github.CurrentUser(cmd.Context()); err == nil {
					defaultHubber = user
				}
			}
			if defaultHubber != "" {
				fmt.Printf("Hubber (default %s): ", defaultHubber)
			} else {
				fmt.Print("Hubber: ")
			}
			hubberText, _ := reader.ReadString('\n')
			hubber = strings.TrimSpace(hubberText)
			if hubber == "" {
				hubber = defaultHubber
			}

			fmt.Print("Ops channel: ")
			opsText, _ := reader.ReadString('\n')
			opsChannel = strings.TrimSpace(opsText)

			fmt.Print("Source of truth path (default docs/SOURCE_OF_TRUTH.md): ")
			sotText, _ := reader.ReadString('\n')
			sotPath = strings.TrimSpace(sotText)
		}

		if sotPath == "" {
			sotPath = "docs/SOURCE_OF_TRUTH.md"
		}

		cfg := config.Config{
			Project: config.ProjectConfig{
				Board: board,
				Owner: owner,
			},
			Agent: config.AgentConfig{
				Hubber: hubber,
				Model:  "gpt-4o",
			},
			Notifications: config.NotificationsConfig{
				Channel:    "slack",
				OpsChannel: opsChannel,
			},
			SourceOfTruth: sotPath,
		}

		if err := config.Write("max-ops.yaml", cfg); err != nil {
			return err
		}

		out := output.New(cmd)
		return out.Print(map[string]string{"config": "max-ops.yaml"})
	},
}

func init() {
	projectInitCmd.Flags().Int("project", 0, "Project board number")
	projectInitCmd.Flags().String("owner", "", "Project owner")
	projectInitCmd.Flags().String("hubber", "", "Developer username")
	projectInitCmd.Flags().String("ops-channel", "", "Slack ops channel")
	projectInitCmd.Flags().String("sot", "", "Source of truth document path")
	projectCmd.AddCommand(projectInitCmd)
}
