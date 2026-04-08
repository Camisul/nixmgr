package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	agessh "github.com/Mic92/ssh-to-age"
	"github.com/camisul/nixmgr/internal/cloudflare"
	"github.com/camisul/nixmgr/internal/config"
	"github.com/camisul/nixmgr/internal/nix"
	"github.com/spf13/cobra"
)

var dryRun bool
var runInstall bool

var addCmd = &cobra.Command{
	Use:   "add <ip-address>",
	Short: "Add a new NixOS host",
	Long: `Add a new NixOS host by:
  1. Picking the next available name from the configured name list
  2. Creating a Cloudflare DNS A record: <name>.<domain> -> <ip>
  3. Scaffolding hosts/<name>/configuration.nix
  4. Adding the host as a flake output in flake.nix`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be done without making any changes")
	addCmd.Flags().BoolVar(&runInstall, "run", false, "Run nixos-anywhere after scaffolding and flake update")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	ip := args[0]
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfgPath := filepath.Join(root, "nixmgr.toml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	hostsDir := filepath.Join(root, "hosts")
	name, err := config.PickName(cfg, hostsDir)
	if err != nil {
		return err
	}

	fqdn := fmt.Sprintf("%s.%s", name, cfg.Domain)
	fmt.Printf("Selected name: %s\n", name)
	fmt.Printf("FQDN:          %s\n", fqdn)
	fmt.Printf("IP:            %s\n", ip)
	fmt.Println()

	if dryRun {
		fmt.Println("[dry-run] Would create DNS record:", fqdn, "->", ip)
		fmt.Printf("[dry-run] Would scaffold hosts/%s/configuration.nix\n", name)
		fmt.Printf("[dry-run] Would add nixosConfigurations.%s to flake.nix\n", name)
		if cfg.Sops.Enabled {
			fmt.Println("[dry-run] Would enable sops-nix integration for this host")
			fmt.Printf("[dry-run] Would fetch machine age key via ssh root@%s cat /etc/ssh/ssh_host_ed25519_key.pub\n", ip)
			fmt.Println("[dry-run] Would update .sops.yaml machine keys")
		}
		if runInstall {
			fmt.Printf("[dry-run] Would run: nixos-anywhere --flake .#%s root@%s\n", name, ip)
		}
		return nil
	}

	// Step 1: Cloudflare DNS
	fmt.Printf("Creating DNS record %s -> %s ... ", fqdn, ip)
	cf, err := cloudflare.NewClient()
	if err != nil {
		return err
	}
	if err := cf.CreateARecord(name, cfg.Domain, ip); err != nil {
		return fmt.Errorf("cloudflare: %w", err)
	}
	fmt.Println("done")

	// Step 2: Scaffold host configuration
	fmt.Printf("Scaffolding hosts/%s/configuration.nix ... ", name)
	if err := nix.ScaffoldHost(hostsDir, name, cfg.Sops.Enabled); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	fmt.Println("done")

	// Step 3: Add flake output
	flakePath := filepath.Join(root, "flake.nix")
	fmt.Printf("Adding flake output nixosConfigurations.%s ... ", name)
	if err := nix.AddFlakeOutput(flakePath, name, cfg.Sops.Enabled); err != nil {
		return fmt.Errorf("flake: %w", err)
	}
	fmt.Println("done")

	if runInstall {
		fmt.Printf("Running nixos-anywhere for %s ...\n", name)
		nixosCmd := exec.Command("nixos-anywhere", "--flake", fmt.Sprintf(".#%s", name), fmt.Sprintf("root@%s", ip))
		nixosCmd.Dir = root
		nixosCmd.Stdout = os.Stdout
		nixosCmd.Stderr = os.Stderr
		nixosCmd.Stdin = os.Stdin
		if err := nixosCmd.Run(); err != nil {
			return fmt.Errorf("running nixos-anywhere: %w", err)
		}
	}

	if cfg.Sops.Enabled {
		fmt.Printf("Fetching machine age identity from root@%s ... ", ip)
		machineAge, err := fetchMachineAgeIdentity(ip)
		if err != nil {
			return fmt.Errorf("fetching machine age identity: %w", err)
		}
		fmt.Println("done")

		sopsPath := filepath.Join(root, ".sops.yaml")
		fmt.Print("Updating .sops.yaml machine keys ... ")
		if err := nix.UpdateSopsMachineKeys(sopsPath, name, machineAge); err != nil {
			return fmt.Errorf("sops: %w", err)
		}
		fmt.Println("done")
	}

	fmt.Println()
	fmt.Printf("Host '%s' added successfully.\n", name)
	fmt.Println()
	if runInstall {
		fmt.Println("nixos-anywhere completed successfully.")
		fmt.Println("Next steps:")
		fmt.Printf("  1. Edit hosts/%s/configuration.nix to customize the host\n", name)
	} else {
		fmt.Println("Next steps:")
		fmt.Printf("  1. Edit hosts/%s/configuration.nix to customize the host\n", name)
		fmt.Printf("  2. Deploy with: nixos-anywhere --flake .#%s root@%s\n", name, ip)
	}

	return nil
}

// findProjectRoot walks up from cwd looking for nixmgr.toml
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "nixmgr.toml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("nixmgr.toml not found (searched upward from cwd)")
		}
		dir = parent
	}
}

func fetchMachineAgeIdentity(ip string) (string, error) {
	sshCmd := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		fmt.Sprintf("root@%s", ip),
		"cat", "/etc/ssh/ssh_host_ed25519_key.pub",
	)

	pubKey, err := sshCmd.Output()
	if err != nil {
		return "", fmt.Errorf("ssh host key fetch failed: %w", err)
	}

	ageRecipient, err := agessh.SSHPublicKeyToAge(pubKey)
	if err != nil {
		return "", fmt.Errorf("ssh-to-age conversion failed: %w", err)
	}

	age := strings.TrimSpace(*ageRecipient)
	if age == "" {
		return "", fmt.Errorf("ssh-to-age returned empty recipient")
	}

	return age, nil
}
