package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/models/opinit_bots"
	"github.com/initia-labs/weave/service"
)

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func ValidateOPinitOptionalBotNameArgs(_ *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("expected zero or one argument, got %d", len(args))
	}
	if len(args) == 1 && !contains([]string{"executor", "challenger"}, args[0]) {
		return fmt.Errorf("invalid bot name '%s'. Valid options are: [executor, challenger]", args[0])
	}
	return nil
}

func ValidateOPinitBotNameArgs(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument, got %d", len(args))
	}
	if !contains([]string{"executor", "challenger"}, args[0]) {
		return fmt.Errorf("invalid bot name '%s'. Valid options are: [executor, challenger]", args[0])
	}
	return nil
}

func OPInitBotsCommand() *cobra.Command {
	shortDescription := "OPInit bots subcommands"
	cmd := &cobra.Command{
		Use:                        "opinit",
		Short:                      shortDescription,
		Long:                       fmt.Sprintf("%s.\n\n%s", shortDescription, OPinitBotsHelperText),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(OPInitBotsKeysSetupCommand())
	cmd.AddCommand(OPInitBotsInitCommand())
	cmd.AddCommand(OPInitBotsStartCommand())
	cmd.AddCommand(OPInitBotsStopCommand())
	cmd.AddCommand(OPInitBotsRestartCommand())
	cmd.AddCommand(OPInitBotsLogCommand())
	cmd.AddCommand(OPInitBotsResetCommand())

	return cmd
}

type HomeConfig struct {
	MinitiaHome string
	OPInitHome  string
	UserHome    string
}

func RunOPInit(nextModelFunc func(ctx context.Context) (tea.Model, error), homeConfig HomeConfig) (tea.Model, error) {
	// Initialize the context with OPInitBotsState
	ctx := weavecontext.NewAppContext(opinit_bots.NewOPInitBotsState())
	ctx = weavecontext.SetMinitiaHome(ctx, homeConfig.MinitiaHome)
	ctx = weavecontext.SetOPInitHome(ctx, homeConfig.OPInitHome)

	// Start the program
	if finalModel, err := tea.NewProgram(
		opinit_bots.NewEnsureOPInitBotsBinaryLoadingModel(
			ctx,
			func(nextCtx context.Context) (tea.Model, error) {
				return opinit_bots.ProcessMinitiaConfig(nextCtx, nextModelFunc)
			},
		), tea.WithAltScreen(),
	).Run(); err != nil {
		fmt.Println("Error running program:", err)
		return finalModel, err
	} else {
		fmt.Println(finalModel.View())
		return finalModel, nil
	}
}

func OPInitBotsKeysSetupCommand() *cobra.Command {
	shortDescription := "Setup keys for OPInit bots"
	setupCmd := &cobra.Command{
		Use:   "setup-keys",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n%s", shortDescription, OPinitBotsHelperText),
		RunE: func(cmd *cobra.Command, args []string) error {
			analytics.TrackRunEvent(cmd, args, analytics.SetupOPinitKeysFeature, analytics.NewEmptyEvent())
			minitiaHome, _ := cmd.Flags().GetString(FlagMinitiaHome)
			opInitHome, _ := cmd.Flags().GetString(FlagOPInitHome)

			userHome, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("error getting user home directory: %v", err)
			}

			_, err = RunOPInit(opinit_bots.NewSetupBotCheckbox, HomeConfig{
				MinitiaHome: minitiaHome,
				OPInitHome:  opInitHome,
				UserHome:    userHome,
			})

			return err
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("cannot get user home directory: %v", err))
	}

	setupCmd.Flags().String(FlagMinitiaHome, filepath.Join(homeDir, common.MinitiaDirectory), "Rollup application directory to fetch artifacts from if existed")
	setupCmd.Flags().String(FlagOPInitHome, filepath.Join(homeDir, common.OPinitDirectory), "OPInit bots home directory")

	return setupCmd
}

func validateConfigFlags(args []string, configPath, keyFilePath string, isGenerateKeyFile bool) error {
	if configPath != "" {
		if len(args) == 0 {
			return fmt.Errorf("bot name <executor|challenger> is required as an argument")
		}
		if keyFilePath != "" && isGenerateKeyFile {
			return fmt.Errorf("invalid configuration: both --generate-key-file and --key-file cannot be set at the same time")
		}
		if keyFilePath == "" && !isGenerateKeyFile {
			return fmt.Errorf("invalid configuration: if --with-config is set, either --generate-key-file or --key-file must be provided")
		}
		if !weaveio.FileOrFolderExists(configPath) {
			return fmt.Errorf("the provided --with-config does not exist: %s", configPath)
		}
	} else {
		// If configPath is empty, neither --generate-key-file nor isGenerateKeyFile should be set
		if keyFilePath != "" || isGenerateKeyFile {
			return fmt.Errorf("invalid configuration: if --with-config is not set, neither --generate-key-file nor --key-file should be provided")
		}
	}

	return nil
}

func handleWithConfig(cmd *cobra.Command, userHome, opInitHome, configPath, keyFilePath string, args []string, force, isGenerateKeyFile bool) error {
	botName := args[0]
	if botName != "executor" && botName != "challenger" {
		return fmt.Errorf("bot name '%s' is not recognized. Allowed values are 'executor' or 'challenger'", botName)
	}

	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var keyFile *weaveio.KeyFile
	if isGenerateKeyFile {
		keyPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("%s.%s.keyfile", common.OpinitGeneratedKeyFilename, botName))
		keyFile, err = opinit_bots.GenerateMnemonicKeyfile(botName)
		if err != nil {
			return fmt.Errorf("error generating keyfile: %v", err)
		}
		err = keyFile.Write(keyPath)
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		fmt.Printf("Key file successfully generated. You can find it at: %s\n", keyPath)
	} else {
		if !weaveio.FileOrFolderExists(keyFilePath) {
			return fmt.Errorf("key file is missing at path: %s", keyFilePath)
		}

		// Read and unmarshal key file data
		keyFile, err = readAndUnmarshalKeyFile(keyFilePath)
		if err != nil {
			return err
		}
	}
	// Handle existing opInitHome directory
	if err := handleExistingOpInitHome(opInitHome, botName, force); err != nil {
		return err
	}

	// Process bot config based on arguments
	if len(args) != 1 {
		return fmt.Errorf("please specify bot name")
	}

	return initializeBotWithConfig(cmd, fileData, keyFile, opInitHome, userHome, botName)
}

// readAndUnmarshalKeyFile read and unmarshal the key file into the KeyFile struct
func readAndUnmarshalKeyFile(keyFilePath string) (*weaveio.KeyFile, error) {
	fileData, err := os.ReadFile(keyFilePath)
	if err != nil {
		return &weaveio.KeyFile{}, err
	}

	var keyFile *weaveio.KeyFile
	err = json.Unmarshal(fileData, &keyFile)
	return keyFile, err
}

// handleExistingOpInitHome handle the case where the opInitHome directory exists
func handleExistingOpInitHome(opInitHome string, botName string, force bool) error {
	if weaveio.FileOrFolderExists(opInitHome) {
		if force {
			// delete db
			dbPath := filepath.Join(opInitHome, fmt.Sprintf("%s.db", botName))
			if weaveio.FileOrFolderExists(dbPath) {
				err := weaveio.DeleteDirectory(dbPath)
				if err != nil {
					return fmt.Errorf("failed to delete %s", dbPath)
				}
			}
		} else {
			return fmt.Errorf("existing %s folder detected. Use --force or -f to override", opInitHome)
		}
	}
	return nil
}

// initializeBotWithConfig initialize a bot based on the provided config
func initializeBotWithConfig(cmd *cobra.Command, fileData []byte, keyFile *weaveio.KeyFile, opInitHome, userHome, botName string) error {
	var err error

	switch botName {
	case "executor":
		var config opinit_bots.ExecutorConfig
		err = json.Unmarshal(fileData, &config)
		if err != nil {
			return err
		}
		err = opinit_bots.InitializeExecutorWithConfig(config, keyFile, opInitHome, userHome)
	case "challenger":
		var config opinit_bots.ChallengerConfig
		err = json.Unmarshal(fileData, &config)
		if err != nil {
			return err
		}
		err = opinit_bots.InitializeChallengerWithConfig(config, keyFile, opInitHome, userHome)
	}
	if err != nil {
		return err
	}

	analytics.TrackCompletedEvent(analytics.SetupOPinitBotFeature)
	fmt.Printf("OPInit bot setup successfully. Config file is saved at %s. Feel free to modify it as needed.\n", filepath.Join(opInitHome, fmt.Sprintf("%s.json", botName)))

	return nil
}

func OPInitBotsInitCommand() *cobra.Command {
	shortDescription := "Initialize an OPinit bot"
	initCmd := &cobra.Command{
		Use:   "init [bot-name]",
		Short: shortDescription,
		Long:  fmt.Sprintf("Initialize an OPinit bot. The argument is optional, as you will be prompted to select a bot if no bot name is provided.\nAlternatively, you can specify a bot name as an argument to skip the selection. Valid options are [executor, challenger].\nExample: weave opinit init executor\n\n%s", OPinitBotsHelperText),
		Args:  ValidateOPinitOptionalBotNameArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			minitiaHome, _ := cmd.Flags().GetString(FlagMinitiaHome)
			opInitHome, _ := cmd.Flags().GetString(FlagOPInitHome)
			force, _ := cmd.Flags().GetBool(FlagForce)
			configPath, _ := cmd.Flags().GetString(FlagWithConfig)
			keyFilePath, _ := cmd.Flags().GetString(FlagKeyFile)
			isGenerateKeyFile, _ := cmd.Flags().GetBool(FlagGenerateKeyFile)
			events := analytics.NewEmptyEvent()

			withConfig := configPath != ""
			if withConfig {
				events.Add(analytics.WithConfigKey, withConfig).
					Add(analytics.KeyFileKey, keyFilePath != "").
					Add(analytics.GenerateKeyfileKey, isGenerateKeyFile)
			}

			var rootProgram func(ctx context.Context) (tea.Model, error)
			// Check if a bot name was provided as an argument
			if len(args) == 1 {
				botName := args[0]
				switch botName {
				case "executor":
					rootProgram = opinit_bots.PrepareExecutorBotKey
				case "challenger":
					rootProgram = opinit_bots.PrepareChallengerBotKey
				default:
					return fmt.Errorf("invalid bot name provided: %s", botName)
				}

				analytics.AppendGlobalEventProperties(map[string]interface{}{
					analytics.BotTypeKey: botName,
				})
			} else {
				// Start the bot selector program if no bot name is provided
				rootProgram = opinit_bots.NewOPInitBotInitSelector
			}

			analytics.TrackRunEvent(cmd, args, analytics.SetupOPinitBotFeature, events)

			userHome, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("error getting user home directory: %v", err)
			}

			err = validateConfigFlags(args, configPath, keyFilePath, isGenerateKeyFile)
			if err != nil {
				return err
			}
			if withConfig {
				return handleWithConfig(cmd, userHome, opInitHome, configPath, keyFilePath, args, force, isGenerateKeyFile)
			}

			_, err = RunOPInit(rootProgram, HomeConfig{
				MinitiaHome: minitiaHome,
				OPInitHome:  opInitHome,
				UserHome:    userHome,
			})

			return err
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("cannot get user home directory: %v", err))
	}

	initCmd.Flags().String(FlagMinitiaHome, filepath.Join(homeDir, common.MinitiaDirectory), "Rollup application directory to fetch artifacts from if existed")
	initCmd.Flags().String(FlagOPInitHome, filepath.Join(homeDir, common.OPinitDirectory), "OPInit bots home directory")
	initCmd.Flags().String(FlagWithConfig, "", "Bypass the interactive setup and initialize the bot by providing a path to a config file. Either --key-file or --generate-key-file has to be specified")
	initCmd.Flags().String(FlagKeyFile, "", "Use this flag to generate the bot keys. Cannot be specified together with --key-file")
	initCmd.Flags().BoolP(FlagForce, "f", false, "Force the setup by deleting the existing .opinit directory if it exists")
	initCmd.Flags().BoolP(FlagGenerateKeyFile, "", false, "Path to key-file.json. Cannot be specified together with --generate-key-file")

	return initCmd
}

func OPInitBotsStartCommand() *cobra.Command {
	startCmd := &cobra.Command{
		Use:     "start [bot-name]",
		Short:   "Start the OPinit bot service.",
		Long:    fmt.Sprintf("Use this command to start the OPinit bot, where the only argument required is the desired bot name.\nValid options are [executor, challenger] eg. weave opinit start executor\n\n%s", OPinitBotsHelperText),
		Args:    ValidateOPinitBotNameArgs,
		PreRunE: isInitiated(service.OPinitExecutor),
		RunE: func(cmd *cobra.Command, args []string) error {
			detach, err := cmd.Flags().GetBool(FlagDetach)
			if err != nil {
				return err
			}

			botName := args[0]
			bot := service.CommandName(botName)
			s, err := service.NewService(bot)
			if err != nil {
				return err
			}

			if detach {
				err = s.Start()
				if err != nil {
					return err
				}
				fmt.Printf("Started the OPinit %[1]s bot. You can see the logs with `weave opinit log %[1]s`\n", botName)
				return nil
			}

			return service.NonDetachStart(s)
		},
	}

	startCmd.Flags().BoolP(FlagDetach, "d", false, "Run the OPinit bot in detached mode")

	return startCmd
}

func OPInitBotsStopCommand() *cobra.Command {
	startCmd := &cobra.Command{
		Use:     "stop [bot-name]",
		Short:   "Stop the OPinit bot service.",
		Long:    fmt.Sprintf("Use this command to stop the OPinit bot, where the only argument required is the desired bot name.\nValid options are [executor, challenger] eg. weave opinit stop challenger\n\n%s", OPinitBotsHelperText),
		Args:    ValidateOPinitBotNameArgs,
		PreRunE: isInitiated(service.OPinitExecutor),
		RunE: func(cmd *cobra.Command, args []string) error {
			botName := args[0]
			bot := service.CommandName(botName)
			s, err := service.NewService(bot)
			if err != nil {
				return err
			}
			err = s.Stop()
			if err != nil {
				return err
			}
			fmt.Printf("Stopped the OPinit %s bot service.\n", botName)
			return nil
		},
	}

	return startCmd
}

func OPInitBotsRestartCommand() *cobra.Command {
	restartCmd := &cobra.Command{
		Use:     "restart [bot-name]",
		Short:   "Restart the OPinit bot service",
		Long:    fmt.Sprintf("Use this command to restart the OPinit bot, where the only argument required is the desired bot name.\nValid options are [executor, challenger] eg. weave opinit restart executor\n\n%s", OPinitBotsHelperText),
		Args:    ValidateOPinitBotNameArgs,
		PreRunE: isInitiated(service.OPinitExecutor),
		RunE: func(cmd *cobra.Command, args []string) error {
			botName := args[0]
			bot := service.CommandName(botName)
			s, err := service.NewService(bot)
			if err != nil {
				return err
			}
			err = s.Restart()
			if err != nil {
				return err
			}
			fmt.Printf("Restart the OPinit %[1]s bot service. You can see the logs with `weave opinit log %[1]s`\n", botName)
			return nil
		},
	}

	return restartCmd
}

func OPInitBotsLogCommand() *cobra.Command {
	shortDescription := "Stream the logs of the OPinit bot service"
	logCmd := &cobra.Command{
		Use:   "log [bot-name]",
		Short: shortDescription,
		Long:  fmt.Sprintf("Stream the logs of the OPinit bot. The only argument required is the desired bot name.\nValid options are [executor, challenger] eg. weave opinit log executor\n\n%s", OPinitBotsHelperText),
		Args:  ValidateOPinitBotNameArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := cmd.Flags().GetInt(FlagN)
			if err != nil {
				return err
			}

			botName := args[0]
			bot := service.CommandName(botName)
			s, err := service.NewService(bot)
			if err != nil {
				return err
			}
			return s.Log(n)
		},
	}

	logCmd.Flags().IntP(FlagN, FlagN, 100, "previous log lines to show")

	return logCmd
}

func OPInitBotsResetCommand() *cobra.Command {
	shortDescription := "Reset an OPinit bot's database"
	resetCmd := &cobra.Command{
		Use:     "reset [bot-name]",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n%s", shortDescription, OPinitBotsHelperText),
		Args:    ValidateOPinitBotNameArgs,
		PreRunE: isInitiated(service.OPinitExecutor),
		RunE: func(cmd *cobra.Command, args []string) error {
			userHome, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("error getting user home directory: %v", err)
			}

			binaryPath := filepath.Join(userHome, common.WeaveDataDirectory, opinit_bots.AppName)
			_, err = cosmosutils.GetBinaryVersion(binaryPath)
			if err != nil {
				return fmt.Errorf("error getting the opinitd binary: %v", err)
			}

			botName := args[0]
			analytics.AppendGlobalEventProperties(map[string]interface{}{
				analytics.BotTypeKey: botName,
			})
			analytics.TrackRunEvent(cmd, args, analytics.ResetOPinitBotFeature, analytics.NewEmptyEvent())
			execCmd := exec.Command(binaryPath, "reset-db", botName)
			if output, err := execCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to reset-db: %v (output: %s)", err, string(output))
			}
			analytics.TrackCompletedEvent(analytics.ResetOPinitBotFeature)
			fmt.Printf("Reset the OPinit %s bot database successfully.\n", botName)
			return nil
		},
	}

	return resetCmd
}
