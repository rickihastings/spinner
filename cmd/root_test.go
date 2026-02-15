package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindConfigFile(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (startDir, homeDir string)
		expectFound bool
		expectPath  string // relative to startDir or homeDir
	}{
		{
			name: "config in current directory",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".spinner.json")
				require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

				return tmpDir, ""
			},
			expectFound: true,
			expectPath:  ".spinner.json",
		},
		{
			name: "config in parent directory",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".spinner.json")
				require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

				childDir := filepath.Join(tmpDir, "subdir")
				require.NoError(t, os.Mkdir(childDir, 0755))

				return childDir, ""
			},
			expectFound: true,
			expectPath:  "../.spinner.json",
		},
		{
			name: "config in grandparent directory",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".spinner.json")
				require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

				childDir := filepath.Join(tmpDir, "a", "b")
				require.NoError(t, os.MkdirAll(childDir, 0755))

				return childDir, ""
			},
			expectFound: true,
			expectPath:  "../../.spinner.json",
		},
		{
			name: "config in home directory fallback",
			setup: func(t *testing.T) (string, string) {
				startDir := t.TempDir()
				homeDir := t.TempDir()
				configPath := filepath.Join(homeDir, ".spinner.json")
				require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

				return startDir, homeDir
			},
			expectFound: true,
			expectPath:  "home/.spinner.json",
		},
		{
			name: "no config file found",
			setup: func(t *testing.T) (string, string) {
				startDir := t.TempDir()
				homeDir := t.TempDir()

				return startDir, homeDir
			},
			expectFound: false,
			expectPath:  "",
		},
		{
			name: "prefers ancestor over home directory",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				homeDir := t.TempDir()

				// Config in parent
				parentConfig := filepath.Join(tmpDir, ".spinner.json")
				require.NoError(t, os.WriteFile(parentConfig, []byte("{}"), 0644))

				// Config in home (should be ignored)
				homeConfig := filepath.Join(homeDir, ".spinner.json")
				require.NoError(t, os.WriteFile(homeConfig, []byte("{}"), 0644))

				childDir := filepath.Join(tmpDir, "subdir")
				require.NoError(t, os.Mkdir(childDir, 0755))

				return childDir, homeDir
			},
			expectFound: true,
			expectPath:  "../.spinner.json",
		},
		{
			name: "empty home directory is handled gracefully",
			setup: func(t *testing.T) (string, string) {
				startDir := t.TempDir()
				return startDir, ""
			},
			expectFound: false,
			expectPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDir, homeDir := tt.setup(t)
			result := findConfigFile(startDir, homeDir)

			if tt.expectFound {
				assert.NotEmpty(t, result, "expected to find config file")

				// Verify the file exists
				_, err := os.Stat(result)
				assert.NoError(t, err, "config file should exist at returned path")

				// Verify it's the expected file
				if tt.expectPath != "home/.spinner.json" {
					expectedPath := filepath.Join(startDir, tt.expectPath)
					expectedPath, _ = filepath.Abs(expectedPath)
					actualPath, _ := filepath.Abs(result)
					assert.Equal(t, expectedPath, actualPath, "should find config at expected location")
				} else {
					expectedPath := filepath.Join(homeDir, ".spinner.json")
					expectedPath, _ = filepath.Abs(expectedPath)
					actualPath, _ := filepath.Abs(result)
					assert.Equal(t, expectedPath, actualPath, "should find config in home directory")
				}
			} else {
				assert.Empty(t, result, "should not find any config file")
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T) (workDir string, cleanup func())
		expectedBackend string
	}{
		{
			name: "ancestor config overrides home config",
			setup: func(t *testing.T) (string, func()) {
				homeDir := t.TempDir()
				repoDir := t.TempDir()
				workDir := filepath.Join(repoDir, "subdir")
				require.NoError(t, os.Mkdir(workDir, 0755))

				// Home config with backend=gcp
				homeConfig := filepath.Join(homeDir, ".spinner.json")
				require.NoError(t, os.WriteFile(homeConfig, []byte(`{"backend":"gcp"}`), 0644))

				// Repo config with backend=docker
				repoConfig := filepath.Join(repoDir, ".spinner.json")
				require.NoError(t, os.WriteFile(repoConfig, []byte(`{"backend":"docker"}`), 0644))

				// Save original HOME and set test HOME
				origHome := os.Getenv("HOME")

				_ = os.Setenv("HOME", homeDir)

				cleanup := func() {
					_ = os.Setenv("HOME", origHome)

					viper.Reset()
				}

				return workDir, cleanup
			},
			expectedBackend: "docker",
		},
		{
			name: "home config used when no ancestor config exists",
			setup: func(t *testing.T) (string, func()) {
				homeDir := t.TempDir()
				workDir := t.TempDir()

				// Only home config exists with backend=gcp
				homeConfig := filepath.Join(homeDir, ".spinner.json")
				require.NoError(t, os.WriteFile(homeConfig, []byte(`{"backend":"gcp"}`), 0644))

				// Save original HOME and set test HOME
				origHome := os.Getenv("HOME")

				_ = os.Setenv("HOME", homeDir)

				cleanup := func() {
					_ = os.Setenv("HOME", origHome)

					viper.Reset()
				}

				return workDir, cleanup
			},
			expectedBackend: "gcp",
		},
		{
			name: "current directory config takes precedence",
			setup: func(t *testing.T) (string, func()) {
				homeDir := t.TempDir()
				workDir := t.TempDir()

				// Home config with backend=gcp
				homeConfig := filepath.Join(homeDir, ".spinner.json")
				require.NoError(t, os.WriteFile(homeConfig, []byte(`{"backend":"gcp"}`), 0644))

				// Current directory config with backend=docker
				currentConfig := filepath.Join(workDir, ".spinner.json")
				require.NoError(t, os.WriteFile(currentConfig, []byte(`{"backend":"docker"}`), 0644))

				// Save original HOME and set test HOME
				origHome := os.Getenv("HOME")

				_ = os.Setenv("HOME", homeDir)

				cleanup := func() {
					_ = os.Setenv("HOME", origHome)

					viper.Reset()
				}

				return workDir, cleanup
			},
			expectedBackend: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, cleanup := tt.setup(t)
			defer cleanup()

			// Save and restore original working directory
			origDir, _ := os.Getwd()
			defer func(dir string) {
				_ = os.Chdir(dir)
			}(origDir)

			// Change to test working directory
			require.NoError(t, os.Chdir(workDir))

			// Reset viper to clean state
			viper.Reset()

			// Simulate the config loading from init()
			viper.SetEnvPrefix("SPINNER")
			viper.AutomaticEnv()

			cwd, _ := os.Getwd()

			home, _ := os.UserHomeDir()
			if configPath := findConfigFile(cwd, home); configPath != "" {
				viper.SetConfigFile(configPath)

				err := viper.ReadInConfig()
				require.NoError(t, err)
			}

			// Verify the backend value matches expected precedence
			backend := viper.GetString("backend")
			assert.Equal(t, tt.expectedBackend, backend, "config precedence should be respected")
		})
	}
}
