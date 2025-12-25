package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/axelyn/envx/internal/exporter"
	"github.com/axelyn/envx/internal/importer"
	"github.com/axelyn/envx/internal/profile"
	"github.com/axelyn/envx/internal/storage"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	envFlag          string
	descFlag         string
	outputFlag       string
	withCommentsFlag bool
	mergeFlag        bool
	dryRunFlag       bool
	overwriteFlag    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "envx",
	Short: "⚡ Lightning-fast environment variable management",
	Long:  `envx - A blazingly fast, local-first CLI tool to manage environment variables across all your projects.`,
}

var initCmd = &cobra.Command{
	Use:   "init <project>",
	Short: "Initialize a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		if err := manager.InitProject(projectName, descFlag, envFlag); err != nil {
			return err
		}

		color.Green("✓ Initialized project '%s' with environment '%s'", projectName, envFlag)
		return nil
	},
}

var setCmd = &cobra.Command{
	Use:   "set <project> <KEY=value>",
	Short: "Set an environment variable",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		// Parse KEY=value pairs
		for _, pair := range args[1:] {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format: %s (expected KEY=value)", pair)
			}

			key := parts[0]
			value := parts[1]

			if err := manager.SetVariable(projectName, envFlag, key, value, descFlag, false); err != nil {
				return err
			}

			color.Green("✓ Set %s=%s", key, value)
		}

		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <project> <KEY>",
	Short: "Get an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		key := args[1]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		variable, err := manager.GetVariable(projectName, envFlag, key)
		if err != nil {
			return err
		}

		fmt.Println(variable.Value)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List all environment variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		variables, err := manager.ListVariables(projectName, envFlag)
		if err != nil {
			return err
		}

		color.Cyan("\n📦 %s (%s)\n", projectName, envFlag)
		fmt.Println()

		if len(variables) == 0 {
			color.Yellow("No variables set")
			return nil
		}

		for key, variable := range variables {
			value := variable.Value
			if variable.IsSecret {
				// Mask secret values
				if len(value) > 8 {
					value = value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
				} else {
					value = strings.Repeat("*", len(value))
				}
			}
			fmt.Printf("%-20s %s\n", key, value)
		}

		fmt.Printf("\n%d variables\n\n", len(variables))
		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export <project>",
	Short: "Export variables to a .env file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		variables, err := manager.ListVariables(projectName, envFlag)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			return fmt.Errorf("no variables to export")
		}

		// Default output path
		if outputFlag == "" {
			outputFlag = ".env"
		}

		// Check if file exists and not overwriting
		if _, err := os.Stat(outputFlag); err == nil && !overwriteFlag {
			color.Yellow("⚠ File '%s' already exists. Use --overwrite to replace it.", outputFlag)
			return fmt.Errorf("file already exists")
		}

		exp := exporter.New()
		if err := exp.ExportToDotenv(variables, outputFlag, withCommentsFlag); err != nil {
			return err
		}

		color.Green("✓ Exported %d variables to %s", len(variables), outputFlag)
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import <project> <file>",
	Short: "Import variables from a .env file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		filePath := args[1]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		// Import the file
		imp := importer.New()
		variables, err := imp.ImportFromDotenv(filePath)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			return fmt.Errorf("no variables found in file")
		}

		// Get existing variables for preview
		existing, _ := manager.ListVariables(projectName, envFlag)
		if existing == nil {
			existing = make(map[string]envx.Variable)
		}

		// Preview changes
		newVars, updatedVars, unchangedVars, err := imp.PreviewImport(filePath, existing)
		if err != nil {
			return err
		}

		// Show preview
		if dryRunFlag {
			color.Cyan("\n📋 Import Preview for %s (%s)\n", projectName, envFlag)

			if len(newVars) > 0 {
				color.Green("\n✓ New variables (%d):", len(newVars))
				for _, key := range newVars {
					fmt.Printf("  + %s\n", key)
				}
			}

			if len(updatedVars) > 0 {
				color.Yellow("\n⚠ Updated variables (%d):", len(updatedVars))
				for _, key := range updatedVars {
					fmt.Printf("  ~ %s\n", key)
				}
			}

			if len(unchangedVars) > 0 {
				color.Blue("\n- Unchanged variables (%d):", len(unchangedVars))
				for _, key := range unchangedVars {
					fmt.Printf("  = %s\n", key)
				}
			}

			fmt.Println("\nRun without --dry-run to apply changes")
			return nil
		}

		// Import variables
		imported := 0
		skipped := 0

		for key, variable := range variables {
			// Skip if merge flag is set and variable exists
			if mergeFlag {
				if _, exists := existing[key]; exists {
					skipped++
					continue
				}
			}

			if err := manager.SetVariable(projectName, envFlag, key, variable.Value, "", false); err != nil {
				color.Yellow("⚠ Failed to import %s: %v", key, err)
				continue
			}
			imported++
		}

		color.Green("✓ Imported %d variables", imported)
		if skipped > 0 {
			color.Yellow("⚠ Skipped %d existing variables (merge mode)", skipped)
		}

		return nil
	},
}

var templateCmd = &cobra.Command{
	Use:   "template <project>",
	Short: "Generate a template file (without values)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		store, err := storage.New()
		if err != nil {
			return err
		}

		manager := profile.New(store)

		if envFlag == "" {
			envFlag = "development"
		}

		variables, err := manager.ListVariables(projectName, envFlag)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			return fmt.Errorf("no variables to export")
		}

		exp := exporter.New()
		if err := exp.ExportTemplate(variables, outputFlag); err != nil {
			return err
		}

		if outputFlag != "" {
			color.Green("✓ Template exported to %s", outputFlag)
		}

		return nil
	},
}

func init() {
	// Init command flags
	initCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")
	initCmd.Flags().StringVarP(&descFlag, "desc", "d", "", "Project description")

	// Set command flags
	setCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")
	setCmd.Flags().StringVarP(&descFlag, "desc", "d", "", "Variable description")

	// Get command flags
	getCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")

	// List command flags
	listCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")

	// Export command flags
	exportCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")
	exportCmd.Flags().StringVarP(&outputFlag, "output", "o", ".env", "Output file path")
	exportCmd.Flags().BoolVar(&withCommentsFlag, "with-comments", false, "Include descriptions as comments")
	exportCmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Overwrite existing file")

	// Import command flags
	importCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")
	importCmd.Flags().BoolVar(&mergeFlag, "merge", false, "Don't overwrite existing variables")
	importCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview changes without applying")

	// Template command flags
	templateCmd.Flags().StringVarP(&envFlag, "env", "e", "", "Environment name")
	templateCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path (default: stdout)")

	// Add commands to root
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(templateCmd)
}