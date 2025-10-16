# Git Settings UI Integration

## Settings Panel Layout

The Git configuration has been integrated into the Settings panel under the "Data" section. Here's how it appears:

### Data Settings Section

```
┌─────────────────────────────────────────────────────────────┐
│ Data                                                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ Workspace path                                              │
│ The absolute path to the workspace folder                   │
│ [________________________________]                         │
│                                                             │
│ ─────────────────────────────────────────────────────────── │
│ Version Control                                              │
│ ─────────────────────────────────────────────────────────── │
│                                                             │
│ ☐ Enable Git                                                │
│ Enable Git version control for your workspace data          │
│                                                             │
│ Remote URL                                                  │
│ Git remote repository URL (e.g., https://github.com/...)   │
│ [________________________________]                         │
│                                                             │
│ Username                                                    │
│ Git username for authentication                             │
│ [____________________]                                      │
│                                                             │
│ Token                                                       │
│ Git token or password for authentication                   │
│ [____________________]                                      │
│                                                             │
│ Branch                                                      │
│ Git branch to use (default: main)                          │
│ [__________]                                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Behavior

### When Git is Disabled (Default)
- Only the "Enable Git" checkbox is visible
- All other Git fields are hidden
- Uses filesystem backend

### When Git is Enabled
- All Git configuration fields become visible
- User can configure:
  - **Remote URL**: Git repository URL (optional)
  - **Username**: Git username for authentication
  - **Token**: Git token/password for authentication
  - **Branch**: Git branch to use (defaults to "main")

### Dynamic Visibility
- Git fields are shown/hidden based on the "Enable Git" checkbox
- Uses the `SetVisibleWhen` functionality to control visibility
- Changes are applied immediately when the checkbox is toggled

## Configuration Flow

1. **User opens Settings** → Data section shows workspace path and Git toggle
2. **User enables Git** → Git configuration fields appear
3. **User configures Git settings** → Fields are validated and stored
4. **User saves settings** → Configuration is persisted to global config
5. **Application restarts** → Uses Git backend if enabled, filesystem if disabled

## Integration Points

- **Configuration Storage**: Stored in `GlobalConfig.Spec.Data.Git`
- **Backend Selection**: `ui/base.go` checks Git configuration and initializes appropriate repository
- **Settings UI**: `ui/pages/settings/view.go` renders the Git configuration fields
- **Domain Model**: `internal/domain/config.go` defines the Git configuration structure

## Example Configuration

```yaml
data:
  workspacePath: "/Users/username/chapar-workspace"
  git:
    enabled: true
    remoteUrl: "https://github.com/team/chapar-workspace.git"
    username: "team-member"
    token: "ghp_xxxxxxxxxxxx"
    branch: "main"
```

This provides a seamless way for users to enable and configure Git version control for their Chapar workspace data directly through the application's settings interface.
