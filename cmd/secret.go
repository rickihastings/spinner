package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rickihastings/spinner/internal/secret"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// secretCmd is the production secret command using a real EncryptedFileStore.
var secretCmd = newSecretCommand(nil)

func init() {
	rootCmd.AddCommand(secretCmd)
}

// storeFactory creates a Store. If nil, uses the default EncryptedFileStore.
type storeFactory func() secret.Store

// newSecretCommand creates the `spinner secret` command tree.
// Pass a non-nil storeFactory for testing; nil uses the real encrypted store.
func newSecretCommand(sf storeFactory) *cobra.Command {
	if sf == nil {
		sf = func() secret.Store {
			return secret.NewEncryptedFileStore("", passphraseFromEnvOrPrompt)
		}
	}

	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets in the encrypted store",
		Long: `Manage secrets in the encrypted store

COMMANDS:
  set <KEY>       Store a secret (prompts for value or use --value)
  list            List stored secret key names
  delete <KEY>    Remove a secret from the store`,
	}

	cmd.AddCommand(newSecretSetCommand(sf))
	cmd.AddCommand(newSecretListCommand(sf))
	cmd.AddCommand(newSecretDeleteCommand(sf))
	cmd.AddCommand(newSecretInjectCommand())

	return cmd
}

// newSecretSetCommand creates the `spinner secret set` subcommand.
func newSecretSetCommand(sf storeFactory) *cobra.Command {
	var valueFlag string

	cmd := &cobra.Command{
		Use:   "set <KEY>",
		Short: "Store a secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := valueFlag

			if value == "" {
				// Prompt for value with hidden input
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enter value for %s: ", key)

				raw, err := readPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("reading secret value: %w", err)
				}

				_, _ = fmt.Fprintln(cmd.OutOrStdout()) // newline after hidden input

				value = strings.TrimSpace(string(raw))
			}

			if value == "" {
				return fmt.Errorf("secret value must not be empty")
			}

			store := sf()
			if err := store.Set(key, value); err != nil {
				return fmt.Errorf("storing secret: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Secret %q stored\n", key)

			return nil
		},
	}

	cmd.Flags().StringVar(&valueFlag, "value", "", "Secret value (alternative to interactive prompt)")

	return cmd
}

// newSecretListCommand creates the `spinner secret list` subcommand.
func newSecretListCommand(sf storeFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored secret key names",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := sf()

			keys, err := store.List()
			if err != nil {
				return fmt.Errorf("listing secrets: %w", err)
			}

			if len(keys) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No secrets stored")
				return nil
			}

			for _, k := range keys {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), k)
			}

			return nil
		},
	}
}

// newSecretDeleteCommand creates the `spinner secret delete` subcommand.
func newSecretDeleteCommand(sf storeFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <KEY>",
		Short: "Remove a secret from the store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			store := sf()

			if err := store.Delete(key); err != nil {
				return fmt.Errorf("deleting secret: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Secret %q deleted\n", key)

			return nil
		},
	}
}

// defaultBlobPath is the default path for the encrypted secrets blob inside containers.
var defaultBlobPath = "/run/spinner/secrets.enc"

// osExit is a package-level variable for testing.
var osExit = os.Exit

// newSecretInjectCommand creates the `spinner secret inject -- <command>` subcommand.
func newSecretInjectCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "inject -- <command> [args...]",
		Short:              "Decrypt secrets blob and run a command with secrets as env vars",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Strip leading "--" if present
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}

			if len(args) == 0 {
				return fmt.Errorf("missing command argument: usage: spinner secret inject -- <command> [args...]")
			}

			passphrase, err := passphraseFromEnvOrPrompt()
			if err != nil {
				return fmt.Errorf("getting passphrase: %w", err)
			}

			secrets, err := secret.DecryptBlob(defaultBlobPath, passphrase)
			if err != nil {
				return fmt.Errorf("decrypting secrets blob: %w", err)
			}

			child := exec.Command(args[0], args[1:]...)

			child.Env = append(os.Environ(), secretsToEnvSlice(secrets)...)
			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr

			if err := child.Run(); err != nil {
				var exitErr *exec.ExitError
				if ok := errors.As(err, &exitErr); ok {
					if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
						osExit(status.ExitStatus())
					}
				}

				return fmt.Errorf("running command: %w", err)
			}

			return nil
		},
	}
}

// secretsToEnvSlice converts a secrets map to KEY=VALUE environment variable strings.
func secretsToEnvSlice(secrets map[string]string) []string {
	env := make([]string, 0, len(secrets))
	for k, v := range secrets {
		env = append(env, k+"="+v)
	}

	return env
}

// readPassword is a package-level variable for testing.
var readPassword = term.ReadPassword

// cached passphrase from a previous interactive prompt within this process.
var cachedPassphrase string

// passphraseFromEnvOrPrompt returns the passphrase from SPINNER_SECRET_PASSPHRASE
// env var, or prompts the user interactively. The interactive result is cached
// for subsequent calls within the same process to avoid repeated prompts.
func passphraseFromEnvOrPrompt() (string, error) {
	if p := os.Getenv("SPINNER_SECRET_PASSPHRASE"); p != "" {
		return p, nil
	}

	if cachedPassphrase != "" {
		return cachedPassphrase, nil
	}

	fmt.Fprint(os.Stderr, "Enter passphrase: ")

	raw, err := readPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}

	fmt.Fprintln(os.Stderr) // newline after hidden input

	p := strings.TrimSpace(string(raw))
	if p == "" {
		return "", fmt.Errorf("passphrase must not be empty")
	}

	cachedPassphrase = p

	return p, nil
}
