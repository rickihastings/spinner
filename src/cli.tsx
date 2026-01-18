#!/usr/bin/env node
import React from 'react';
import { render } from 'ink';
import { App } from './App.js';

const args = process.argv.slice(2);
const command = args.find((arg) => !arg.startsWith('--'));
const flags: Record<string, string | boolean> = {};

// Convert kebab-case to camelCase
function toCamelCase(str: string): string {
  return str.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}

for (let i = 0; i < args.length; i++) {
  const arg = args[i];
  if (arg.startsWith('--')) {
    const key = toCamelCase(arg.slice(2));
    const nextArg = args[i + 1];
    if (nextArg && !nextArg.startsWith('--')) {
      flags[key] = nextArg;
      i++;
    } else {
      flags[key] = true;
    }
  }
}

render(
  <App
    command={command}
    flags={
      flags as {
        name?: string;
        nodeVersion?: string;
        jvmUrl?: string;
        image?: string;
        repo?: string;
        help?: boolean;
        version?: boolean;
      }
    }
  />,
);
