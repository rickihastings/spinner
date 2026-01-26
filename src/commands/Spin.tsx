import React, { useState, useEffect } from 'react';
import { Text, Box } from 'ink';
import {
  validatePrerequisites,
  generateContainerName,
  buildDockerRunCommand,
  executeDockerRun,
  verifyContainerStatus,
  checkContainerExists,
  restartContainer,
  removeContainer,
  type SpinConfig,
} from '../utils/docker.js';

export interface SpinProps {
  image: string;
  repo: string;
  prompt?: string;
  branch?: string;
  maxIterations?: string;
  recreate?: boolean;
}

export const Spin: React.FC<SpinProps> = ({
  image,
  repo,
  prompt,
  branch,
  maxIterations,
  recreate,
}) => {
  const [status, setStatus] = useState<'validating' | 'creating' | 'success' | 'error'>(
    'validating',
  );
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [containerName, setContainerName] = useState<string>('');
  const [warnings, setWarnings] = useState<string[]>([]);
  const [containerAction, setContainerAction] = useState<'created' | 'reused' | 'restarted'>(
    'created',
  );

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
        const generatedName = generateContainerName(config);
        setContainerName(generatedName);

        setStatus('creating');
        setWarnings(validationResult.warnings);

        // Check if container already exists
        const containerStatus = checkContainerExists(generatedName);

        // Handle --recreate flag: remove existing container and create fresh
        if (recreate && containerStatus !== 'none') {
          const removeResult = removeContainer(generatedName);
          if (!removeResult.success) {
            throw new Error(removeResult.error);
          }
        }

        if (recreate || containerStatus === 'none') {
          // Create new container
          const dockerArgs = buildDockerRunCommand(
            config,
            generatedName,
            validationResult.hasNpmrc,
          );

          const runResult = executeDockerRun(dockerArgs, generatedName);
          if (!runResult.success) {
            throw new Error(runResult.error);
          }

          // Verify container is running
          const statusResult = verifyContainerStatus(generatedName);
          if (!statusResult.success) {
            throw new Error(statusResult.error);
          }

          setContainerAction('created');
          setStatus('success');
        } else if (containerStatus === 'running') {
          // Reuse running container
          setContainerAction('reused');
          setStatus('success');
        } else if (containerStatus === 'stopped') {
          // Restart stopped container
          const restartResult = restartContainer(generatedName);
          if (!restartResult.success) {
            throw new Error(restartResult.error);
          }

          // Verify container is running after restart
          const statusResult = verifyContainerStatus(generatedName);
          if (!statusResult.success) {
            throw new Error(statusResult.error);
          }

          setContainerAction('restarted');
          setStatus('success');
        }
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
  }, [image, repo, prompt, branch, maxIterations, recreate]);

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
    const actionMessage =
      containerAction === 'created'
        ? `✓ Container created successfully: ${containerName}`
        : containerAction === 'restarted'
          ? `✓ Container restarted: ${containerName}`
          : `✓ Reusing running container: ${containerName}`;

    const managementNote =
      containerAction !== 'created'
        ? 'Note: Reusing existing container. Use --recreate flag to force recreation.'
        : null;

    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites validated</Text>
        {warnings.map((warning, idx) => (
          <Text key={idx} color="yellow">
            ⚠ Warning: {warning}
          </Text>
        ))}
        <Text color="green">{actionMessage}</Text>
        {managementNote && (
          <>
            <Text></Text>
            <Text color="cyan">{managementNote}</Text>
          </>
        )}
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
