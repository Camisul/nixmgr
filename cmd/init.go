package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/soyboy/nixmgr/internal/nix"
	"github.com/spf13/cobra"
)

var initDomain string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new nixmgr project in the current directory",
	Long: `Initialize a new nixmgr-managed flakes repository in the current directory.

This creates:
  - flake.nix    Nix flake with nixpkgs and disko inputs
  - nixmgr.toml  Configuration file with domain and name pool
  - hosts/       Empty directory for host configurations

If the directory is not already a git repository, git init is run automatically.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initDomain, "domain", "example.com", "Domain for host subdomains")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Check if already initialized
	tomlPath := filepath.Join(cwd, "nixmgr.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		return fmt.Errorf("nixmgr.toml already exists — project is already initialized")
	}

	// Scaffold the project
	fmt.Println("Initializing nixmgr project...")
	fmt.Println()

	if err := nix.InitProject(cwd, initDomain); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	fmt.Println("  created flake.nix")
	fmt.Println("  created nixmgr.toml")
	fmt.Println("  created hosts/")

	// Git init if needed
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println()
		fmt.Print("Initializing git repository ... ")
		if err := exec.Command("git", "init").Run(); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
		fmt.Println("done")
	}

	fmt.Println()
	fmt.Println("Project initialized. Next steps:")
	fmt.Printf("  1. Edit nixmgr.toml to set your domain (currently %q)\n", initDomain)
	fmt.Println("  2. Add hosts with: nixmgr add <ip-address>")

	return nil
}
