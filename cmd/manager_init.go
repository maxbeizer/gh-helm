package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var managerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a helm-manager.toml in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		hubber, _ := cmd.Flags().GetString("hubber")
		reader := bufio.NewReader(os.Stdin)

		if hubber == "" {
			if user, err := github.CurrentUser(cmd.Context()); err == nil {
				hubber = user
			}
		}
		fmt.Printf("Hubber (default %s): ", hubber)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text != "" {
			hubber = text
		}

		projects := []config.ManagerProject{}
		fmt.Print("Number of projects: ")
		projText, _ := reader.ReadString('\n')
		projText = strings.TrimSpace(projText)
		projCount, _ := strconv.Atoi(projText)
		for i := 0; i < projCount; i++ {
			fmt.Printf("Project %d owner: ", i+1)
			ownerText, _ := reader.ReadString('\n')
			fmt.Printf("Project %d board number: ", i+1)
			boardText, _ := reader.ReadString('\n')
			fmt.Printf("Project %d name: ", i+1)
			nameText, _ := reader.ReadString('\n')
			boardNum, _ := strconv.Atoi(strings.TrimSpace(boardText))
			projects = append(projects, config.ManagerProject{
				Owner: strings.TrimSpace(ownerText),
				Board: boardNum,
				Name:  strings.TrimSpace(nameText),
			})
		}

		team := []config.TeamMember{}
		fmt.Print("Number of team members: ")
		teamText, _ := reader.ReadString('\n')
		teamText = strings.TrimSpace(teamText)
		teamCount, _ := strconv.Atoi(teamText)
		for i := 0; i < teamCount; i++ {
			fmt.Printf("Team member %d handle: ", i+1)
			handleText, _ := reader.ReadString('\n')
			fmt.Printf("Team member %d 1-1 repo (owner/repo): ", i+1)
			repoText, _ := reader.ReadString('\n')
			fmt.Printf("Team member %d pillars (comma-separated): ", i+1)
			pillarsText, _ := reader.ReadString('\n')
			team = append(team, config.TeamMember{
				Handle:    strings.TrimSpace(handleText),
				OneOneRepo: strings.TrimSpace(repoText),
				Pillars:   splitCSV(pillarsText),
			})
		}

		pillars := map[string]config.PillarConfig{}
		fmt.Print("Number of pillars: ")
		pillarText, _ := reader.ReadString('\n')
		pillarText = strings.TrimSpace(pillarText)
		pillarCount, _ := strconv.Atoi(pillarText)
		for i := 0; i < pillarCount; i++ {
			fmt.Printf("Pillar %d key (e.g. reliability): ", i+1)
			keyText, _ := reader.ReadString('\n')
			key := strings.TrimSpace(keyText)
			fmt.Printf("Pillar %d description: ", i+1)
			descText, _ := reader.ReadString('\n')
			fmt.Printf("Pillar %d signals (comma-separated): ", i+1)
			signalsText, _ := reader.ReadString('\n')
			fmt.Printf("Pillar %d repos (comma-separated): ", i+1)
			reposText, _ := reader.ReadString('\n')
			fmt.Printf("Pillar %d labels (comma-separated): ", i+1)
			labelsText, _ := reader.ReadString('\n')
			pillars[key] = config.PillarConfig{
				Description: strings.TrimSpace(descText),
				Signals:     splitCSV(signalsText),
				Repos:       splitCSV(reposText),
				Labels:      splitCSV(labelsText),
			}
		}

		cfg := config.ManagerConfig{
			Manager:  config.ManagerSettings{Hubber: hubber},
			Projects: projects,
			Team:     team,
			Pillars:  pillars,
			Notifications: config.NotificationsConfig{
				Channel:    "slack",
				OpsChannel: "#engineering-ops",
			},
		}

		if err := config.WriteManager("helm-manager.toml", cfg); err != nil {
			return err
		}

		out := output.New(cmd)
		return out.Print(map[string]string{"config": "helm-manager.toml"})
	},
}

func splitCSV(input string) []string {
	parts := strings.Split(strings.TrimSpace(input), ",")
	values := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func init() {
	managerInitCmd.Flags().String("hubber", "", "Manager username")
	managerCmd.AddCommand(managerInitCmd)
}
