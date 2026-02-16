package cmd

import (
	"fmt"
	"os"
	"strings"

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
				fmt.Fprintf(cmd.OutOrStdout(), "Enter value for %s: ", key)

				raw, err := readPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("reading secret value: %w", err)
				}

				fmt.Fprintln(cmd.OutOrStdout()) // newline after hidden input
				value = strings.TrimSpace(string(raw))
			}

			if value == "" {
				return fmt.Errorf("secret value must not be empty")
			}

			store := sf()
			if err := store.Set(key, value); err != nil {
				return fmt.Errorf("storing secret: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Secret %q stored\n", key)

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
				fmt.Fprintln(cmd.OutOrStdout(), "No secrets stored")
				return nil
			}

			for _, k := range keys {
				fmt.Fprintln(cmd.OutOrStdout(), k)
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

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Secret %q deleted\n", key)

			return nil
		},
	}
}

// readPassword is a package-level variable for testing.
var readPassword = term.ReadPassword

// passphraseFromEnvOrPrompt returns the passphrase from SPINNER_SECRET_PASSPHRASE
// env var, or prompts the user interactively.
func passphraseFromEnvOrPrompt() (string, error) {
	if p := os.Getenv("SPINNER_SECRET_PASSPHRASE"); p != "" {
		return p, nil
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

	return p, nil
}
