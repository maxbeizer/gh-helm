package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

// parseProjectURL extracts owner and board number from a GitHub Projects URL.
// Supported formats:
//
//	https://github.com/orgs/<owner>/projects/<number>
//	https://github.com/users/<owner>/projects/<number>
func parseProjectURL(raw string) (owner string, board int, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", 0, fmt.Errorf("invalid URL: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// expect: [orgs|users, <owner>, projects, <number>]
	if len(parts) < 4 || parts[2] != "projects" {
		return "", 0, fmt.Errorf("URL must be https://github.com/{orgs|users}/<owner>/projects/<number>")
	}
	owner = parts[1]
	board, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, fmt.Errorf("invalid board number in URL: %w", err)
	}
	return owner, board, nil
}

var projectInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a helm.toml in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		board, _ := cmd.Flags().GetInt("project")
		owner, _ := cmd.Flags().GetString("owner")
		boardURL, _ := cmd.Flags().GetString("board-url")
		username, _ := cmd.Flags().GetString("user")
		model, _ := cmd.Flags().GetString("model")
		maxPerHour, _ := cmd.Flags().GetInt("max-per-hour")
		channel, _ := cmd.Flags().GetString("channel")
		opsChannel, _ := cmd.Flags().GetString("ops-channel")
		webhookURL, _ := cmd.Flags().GetString("webhook-url")
		sotPath, _ := cmd.Flags().GetString("sot")
		statusFilter, _ := cmd.Flags().GetString("status")
		labels, _ := cmd.Flags().GetStringSlice("labels")

		// --board-url overrides --project and --owner
		if boardURL != "" {
			parsedOwner, parsedBoard, err := parseProjectURL(boardURL)
			if err != nil {
				return err
			}
			owner = parsedOwner
			board = parsedBoard
		}

		noFlags := board == 0 && owner == "" && username == "" && opsChannel == "" && sotPath == ""

		if noFlags {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Project board number or URL (blank to create one): ")
			boardText, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			boardText = strings.TrimSpace(boardText)
			if boardText != "" {
				if strings.HasPrefix(boardText, "http") {
					parsedOwner, parsedBoard, err := parseProjectURL(boardText)
					if err != nil {
						return err
					}
					owner = parsedOwner
					board = parsedBoard
				} else {
					parsed, err := strconv.Atoi(boardText)
					if err != nil {
						return fmt.Errorf("invalid project board: %w", err)
					}
					board = parsed
				}
			}

			if owner == "" {
				defaultOwner := ""
				if user, err := github.CurrentUser(cmd.Context()); err == nil {
					defaultOwner = user
				}
				if defaultOwner != "" {
					fmt.Printf("Project owner (default %s): ", defaultOwner)
				} else {
					fmt.Print("Project owner: ")
				}
				ownerText, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				owner = strings.TrimSpace(ownerText)
				if owner == "" {
					owner = defaultOwner
				}
			}

			// Create a new board if none was provided.
			if board == 0 && owner != "" {
				repoName := ""
				if r, err := github.CurrentRepo(cmd.Context()); err == nil {
					repoName = r
				}
				title := repoName
				if title == "" {
					title = "gh-helm project"
				}
				fmt.Fprintf(os.Stdout, "  Creating project board %q for %s...\n", title, owner)
				result, err := github.CreateProject(cmd.Context(), owner, title)
				if err != nil {
					return fmt.Errorf("create project board: %w", err)
				}
				board = result.Number
				fmt.Fprintf(os.Stdout, "  ✅ Created: %s\n", result.URL)
				fmt.Fprintf(os.Stdout, "  Statuses: Ready, In Progress, In Review, Done\n\n")
			}

			defaultUser := ""
			if username == "" {
				if user, err := github.CurrentUser(cmd.Context()); err == nil {
					defaultUser = user
				}
			}
			if defaultUser != "" {
				fmt.Printf("Username (default %s): ", defaultUser)
			} else {
				fmt.Print("Username: ")
			}
			userText, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			username = strings.TrimSpace(userText)
			if username == "" {
				username = defaultUser
			}

			fmt.Print("Ops channel: ")
			opsText, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			opsChannel = strings.TrimSpace(opsText)

			fmt.Println()
			fmt.Println("Choose an AI model:")
			fmt.Println()
			fmt.Println("  1. gpt-4.1       (default) Fast, reliable, good for most tasks")
			fmt.Println("  2. gpt-5                    Higher quality, best overall balance")
			fmt.Println("  3. o3                        Best quality, slowest, complex code")
			fmt.Println("  4. deepseek-r1               Strong at code, good alternative")
			fmt.Println("  5. Other                     Enter a custom model name")
			fmt.Println()
			fmt.Println("  See all models: https://github.com/marketplace/models")
			fmt.Println()
			fmt.Print("Model [1]: ")
			modelText, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			modelChoice := strings.TrimSpace(modelText)
			switch modelChoice {
			case "", "1":
				model = "gpt-4.1"
			case "2":
				model = "gpt-5"
			case "3":
				model = "o3"
			case "4":
				model = "deepseek-r1"
			case "5":
				fmt.Print("Model name: ")
				customModel, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				model = strings.TrimSpace(customModel)
				if model == "" {
					model = "gpt-4.1"
				}
			default:
				// Treat raw input as a model name.
				model = modelChoice
			}

			fmt.Print("Source of truth path (default docs/SOURCE_OF_TRUTH.md): ")
			sotText, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			sotPath = strings.TrimSpace(sotText)
		}

		if sotPath == "" {
			sotPath = "docs/SOURCE_OF_TRUTH.md"
		}

		if username == "" {
			defaultUser := ""
			if user, err := github.CurrentUser(cmd.Context()); err == nil {
				defaultUser = user
			}
			username = defaultUser
		}

		if model == "" {
			model = "gpt-4.1"
		}
		if channel == "" {
			channel = "slack"
		}

		cfg := config.Config{
			Version: config.CurrentConfigVersion,
			Project: config.ProjectConfig{
				Board: board,
				Owner: owner,
			},
			Agent: config.AgentConfig{
				User:       username,
				Model:      model,
				MaxPerHour: maxPerHour,
			},
			Notifications: config.NotificationsConfig{
				Channel:    channel,
				OpsChannel: opsChannel,
				WebhookURL: webhookURL,
			},
			SourceOfTruth: sotPath,
			Filters: config.FiltersConfig{
				Status: statusFilter,
				Labels: labels,
			},
		}

		if err := config.Write("helm.toml", cfg); err != nil {
			return err
		}

		out := output.New(cmd)
		return out.Print(map[string]string{"config": "helm.toml"})
	},
}

func init() {
	projectInitCmd.Flags().Int("project", 0, "Project board number")
	projectInitCmd.Flags().String("owner", "", "Project owner")
	projectInitCmd.Flags().String("board-url", "", "Project board URL (e.g. https://github.com/users/octocat/projects/1)")
	projectInitCmd.Flags().String("user", "", "Developer username")
	projectInitCmd.Flags().String("model", "", "AI model (default gpt-4.1)")
	projectInitCmd.Flags().Int("max-per-hour", 0, "Rate limit for agent actions")
	projectInitCmd.Flags().String("channel", "", "Notification channel (default slack)")
	projectInitCmd.Flags().String("ops-channel", "", "Slack ops channel")
	projectInitCmd.Flags().String("webhook-url", "", "Notification webhook URL")
	projectInitCmd.Flags().String("sot", "", "Source of truth document path")
	projectInitCmd.Flags().String("status", "", "Project board status filter for daemon")
	projectInitCmd.Flags().StringSlice("labels", nil, "Labels filter for issue pickup")
	projectCmd.AddCommand(projectInitCmd)
}
