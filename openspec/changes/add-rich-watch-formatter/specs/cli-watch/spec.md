# cli-watch Specification Delta

## ADDED Requirements

### Requirement: Rich Tool Call Display

The watch formatter SHALL render tool_use blocks from assistant messages as visually distinct single-line
summaries showing the tool name and a parameter summary, rather than hiding them.

#### Scenario: Bash tool call displayed with command

- **WHEN** an assistant message contains a tool_use block with name `Bash` and input `{"command": "brew install tmux"}`
- **THEN** the formatter SHALL render it as `⏺ Bash(brew install tmux)` with the tool name visually emphasized

#### Scenario: Read tool call displayed with file path

- **WHEN** an assistant message contains a tool_use block with name `Read` and input `{"file_path": "/path/to/file.go"}`
- **THEN** the formatter SHALL render it as `⏺ Read(/path/to/file.go)`

#### Scenario: Unknown tool displayed with fallback

- **WHEN** an assistant message contains a tool_use block with an unrecognized tool name
- **THEN** the formatter SHALL render it as `⏺ ToolName(first_param_value)` using the first string field from input, or `⏺ ToolName` if no suitable field exists

#### Scenario: Long parameter value truncated

- **WHEN** a tool parameter summary exceeds 80 characters
- **THEN** the formatter SHALL truncate it and append `...`

#### Scenario: Tool use ID recorded for result correlation

- **WHEN** a tool_use block is rendered
- **THEN** the formatter SHALL record the mapping from `tool_use_id` to `tool_name` for use when the corresponding tool result arrives

### Requirement: Rich Tool Result Display

The watch formatter SHALL render tool_result blocks from user messages as indented output summaries
showing the associated tool name, line count, and error status, rather than skipping them entirely.

#### Scenario: Successful tool result with line count

- **WHEN** a user message contains a tool_result block with text content of 84 lines and `is_error` is false
- **THEN** the formatter SHALL render a header line with the tool name and a summary showing `+84 lines`

#### Scenario: Error tool result

- **WHEN** a user message contains a tool_result block with `is_error` true
- **THEN** the formatter SHALL render the tool name header with an error indicator and include the error message text

#### Scenario: Tool result correlated with invocation

- **WHEN** a tool_result block references a `tool_use_id` that was previously recorded
- **THEN** the formatter SHALL display the corresponding tool name in the result header
- **AND** the formatter SHALL remove the tool_use_id entry from its tracking map

#### Scenario: Tool result with unknown ID

- **WHEN** a tool_result block references a `tool_use_id` that is not in the tracking map
- **THEN** the formatter SHALL render the result with a generic `Tool` label instead of a specific tool name

### Requirement: Rich Markdown Rendering

The watch formatter SHALL render text content from assistant messages using markdown formatting with
support for headings, bold text, bullet points, numbered lists, and syntax-highlighted code blocks.

#### Scenario: Markdown text rendered with formatting

- **WHEN** an assistant message contains text with markdown formatting (headings, bold, lists, code blocks)
- **THEN** the formatter SHALL render the text through a markdown-to-terminal pipeline producing styled output with colors, indentation, and syntax highlighting

#### Scenario: Code blocks rendered with syntax highlighting

- **WHEN** an assistant message contains a fenced code block with a language identifier
- **THEN** the formatter SHALL render the code block with syntax highlighting appropriate for the specified language

#### Scenario: Plain text rendered without modification

- **WHEN** an assistant message contains text without any markdown formatting
- **THEN** the formatter SHALL render the text as-is without adding unwanted formatting artifacts

#### Scenario: ANSI output converted to TUI color tags

- **WHEN** the markdown renderer produces ANSI escape sequences
- **THEN** the formatter SHALL convert them to tview-compatible color tags before returning the formatted string

## MODIFIED Requirements

### Requirement: Log Parsing

The CLI SHALL parse JSON-formatted log entries and display them in human-readable format.

#### Scenario: Parse JSON log entry

- **WHEN** a log entry is in JSON format with timestamp, level, and message fields
- **THEN** the CLI SHALL format it as "[timestamp] [level] message"

#### Scenario: Parse non-JSON log entry

- **WHEN** a log entry is not valid JSON
- **THEN** the CLI SHALL display it as-is without parsing

#### Scenario: Extract iteration from logs

- **WHEN** a log entry contains iteration information (e.g., "Iteration 5/100")
- **THEN** the CLI SHALL update the header iteration counter

#### Scenario: User messages rendered by formatter

- **WHEN** a log entry is a user message containing tool_result blocks
- **THEN** the CLI SHALL pass it to the event formatter for rendering rather than unconditionally skipping it
