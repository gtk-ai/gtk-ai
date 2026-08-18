## gtk-ai — rule-based output filtering

gtk-ai filters Bash, grep, find, ls, git, and MCP tool output before it enters
the context. Depending on the command, it applies truncation, extension grouping,
condensed formatting, or comment line removal.

The hook is active only when the Claude plugin is installed and enabled.
Run `claude plugin install -s user gtk-ai@gtk-ai` if you have not done so.
