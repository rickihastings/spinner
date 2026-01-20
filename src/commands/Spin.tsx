import React, { useState, useEffect } from 'react';
import { Text, Box } from 'ink';
import {
  validatePrerequisites,
  generateContainerName,
  buildDockerRunCommand,
  executeDockerRun,
  verifyContainerStatus,
  type SpinConfig,
} from '../utils/docker.js';

export interface SpinProps {
  image: string;
  repo: string;
  prompt?: string;
  branch?: string;
  maxIterations?: string;
}

export const Spin: React.FC<SpinProps> = ({ image, repo, prompt, branch, maxIterations }) => {
  const [status, setStatus] = useState<'validating' | 'creating' | 'success' | 'error'>(
    'validating',
  );
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [containerName, setContainerName] = useState<string>('');
  const [warnings, setWarnings] = useState<string[]>([]);

  useEffect(() => {
    async function run() {
      try {
        setStatus('validating');

        const config: SpinConfig = {
          image,
          repo,
          prompt,
          branch,
          maxIterations,
        };

        // Validate prerequisites
        const validationResult = validatePrerequisites(config);
        if (!validationResult.valid) {
          throw new Error(validationResult.error);
        }

        // Generate container name
        const generatedName = generateContainerName(repo);
        setContainerName(generatedName);

        setStatus('creating');
        setWarnings(validationResult.warnings);

        // Build docker run command
        const dockerArgs = buildDockerRunCommand(config, generatedName, validationResult.hasNpmrc);

        // Execute docker run
        const runResult = executeDockerRun(dockerArgs, generatedName);
        if (!runResult.success) {
          throw new Error(runResult.error);
        }

        // Verify container is running
        const statusResult = verifyContainerStatus(generatedName);
        if (!statusResult.success) {
          throw new Error(statusResult.error);
        }

        setStatus('success');
      } catch (error) {
        setStatus('error');
        if (error instanceof Error) {
          setErrorMessage(error.message);
        } else {
          setErrorMessage('Unknown error occurred');
        }
      }
    }

    run();
  }, [image, repo, prompt, branch, maxIterations]);

  if (status === 'validating') {
    return (
      <Box flexDirection="column">
        <Text>Validating prerequisites...</Text>
      </Box>
    );
  }

  if (status === 'creating') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites validated</Text>
        {warnings.map((warning, idx) => (
          <Text key={idx} color="yellow">
            ⚠ Warning: {warning}
          </Text>
        ))}
        <Text>Creating container: {containerName}</Text>
        <Text>Cloning repository...</Text>
      </Box>
    );
  }

  if (status === 'success') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites validated</Text>
        {warnings.map((warning, idx) => (
          <Text key={idx} color="yellow">
            ⚠ Warning: {warning}
          </Text>
        ))}
        <Text color="green">✓ Container created successfully: {containerName}</Text>
        <Text></Text>
        <Text>To access: docker exec -it {containerName} bash</Text>
        <Text>To stop: docker stop {containerName}</Text>
        <Text>To remove: docker rm {containerName}</Text>
      </Box>
    );
  }

  // Error state
  process.exitCode = 1;
  return (
    <Box flexDirection="column">
      <Text color="red">✗ Error: {errorMessage}</Text>
    </Box>
  );
};
