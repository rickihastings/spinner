import React from 'react';
import { Text, Box } from 'ink';
import { Setup } from './commands/Setup.js';
import { Spin } from './commands/Spin.js';

export interface AppProps {
  command?: string;
  flags: {
    name?: string;
    baseImage?: string;
    dockerfile?: string;
    image?: string;
    repo?: string;
    prompt?: string;
    branch?: string;
    maxIterations?: string;
    help?: boolean;
    version?: boolean;
  };
}

const HELP_TEXT = `
Spinner - CLI tool for running code in isolated Docker containers

USAGE:
  spinner setup --name <name> [--base-image <image> | --dockerfile <path>]
  spinner spin --image <image> --repo <repo> [--prompt <prompt> --branch <branch> [--max-iterations <num>]]

COMMANDS:
  setup    Build a Docker sandbox image with custom base image or Dockerfile
  spin     Spin up a development container from a pre-built image

SETUP OPTIONS:
  --name <name>              Name for the Docker image (required)
  --base-image <image>       Base Docker image (optional, default: ubuntu:22.04)
  --dockerfile <path>        Path to custom Dockerfile (optional, mutually exclusive with --base-image)

SPIN OPTIONS:
  --image <image>            Docker image name (required)
  --repo <repo>              Git SSH clone URL (required)
  --prompt <prompt>          Prompt string for autonomous implementation (optional)
  --branch <branch>          Branch to work on (optional, uses default branch if not specified)
  --max-iterations <num>     Maximum Ralph loop iterations (optional, default: 100)

GENERAL OPTIONS:
  --help                     Show this help message
  --version                  Show version information

EXAMPLES:
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --base-image ubuntu:22.04
  spinner setup --name node-env --base-image node:20-bullseye
  spinner setup --name custom-env --dockerfile ./Dockerfile.custom
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --prompt "Implement feature X"
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --prompt "Implement feature X" --branch feature/x

NOTES:
  - Setup: Only Ubuntu/Debian-based images are supported (requires apt-get)
  - Setup: The CLI ensures git and claude-code are installed in the final image
  - Setup: If using --dockerfile, the custom Dockerfile is built first and used as base
  - Spin: SSH agent must be running on host system
  - Spin: Containers are persistent and must be manually stopped/removed
`;

export const App: React.FC<AppProps> = ({ command, flags }) => {
  if (flags.help) {
    return <Text>{HELP_TEXT}</Text>;
  }

  if (flags.version) {
    return <Text>Spinner version 0.1.0</Text>;
  }

  if (command === 'setup') {
    if (!flags.name) {
      return (
        <Box flexDirection="column">
          <Text color="red">Error: Missing required flag: --name</Text>
          <Text>
            Usage: spinner setup --name &lt;name&gt; [--base-image &lt;image&gt; | --dockerfile
            &lt;path&gt;]
          </Text>
        </Box>
      );
    }

    if (flags.baseImage && flags.dockerfile) {
      return (
        <Box flexDirection="column">
          <Text color="red">Error: --base-image and --dockerfile are mutually exclusive</Text>
          <Text>Please provide only one of these flags</Text>
        </Box>
      );
    }

    return <Setup name={flags.name} baseImage={flags.baseImage} dockerfile={flags.dockerfile} />;
  }

  if (command === 'spin') {
    if (!flags.image) {
      process.exitCode = 1;
      return (
        <Box flexDirection="column">
          <Text color="red">Error: --image flag is required</Text>
          <Text>Usage: spinner spin --image &lt;image&gt; --repo &lt;repo&gt;</Text>
        </Box>
      );
    }

    if (!flags.repo) {
      process.exitCode = 1;
      return (
        <Box flexDirection="column">
          <Text color="red">Error: --repo flag is required</Text>
          <Text>Usage: spinner spin --image &lt;image&gt; --repo &lt;repo&gt;</Text>
        </Box>
      );
    }

    return (
      <Spin
        image={flags.image}
        repo={flags.repo}
        prompt={flags.prompt}
        branch={flags.branch}
        maxIterations={flags.maxIterations}
      />
    );
  }

  return (
    <Box flexDirection="column">
      <Text color="red">Unknown command: {command || '(none)'}</Text>
      <Text>Run with --help for usage information</Text>
    </Box>
  );
};
