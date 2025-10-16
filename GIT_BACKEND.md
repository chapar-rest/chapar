# Git Backend for Chapar

This document explains how to use Git as a data backend for Chapar, providing version control and collaboration features.

## Configuration

The Git backend can be configured through the application settings. The following configuration options are available:

```yaml
data:
  workspacePath: "/path/to/workspace"
  git:
    enabled: true
    remoteUrl: "https://github.com/username/chapar-workspace.git"
    username: "your-username"
    token: "your-github-token"
    branch: "main"
```

### Configuration Fields

- `enabled`: Enable/disable Git backend (default: false)
- `remoteUrl`: Git remote repository URL (optional)
- `username`: Git username for authentication
- `token`: Git token/password for authentication
- `branch`: Git branch to use (default: "main")

## Features

### Automatic Version Control
- All changes to requests, collections, environments, proto files, and workspaces are automatically committed to Git
- Each operation creates a descriptive commit message
- Full commit history is maintained for all changes

### Remote Synchronization
- When configured with a remote URL, changes can be pushed to and pulled from the remote repository
- Automatic pull before loading data ensures you have the latest changes
- Push after each operation keeps the remote repository up to date

### Collaboration
- Multiple team members can work on the same Chapar workspace
- Git handles merge conflicts and change tracking
- Full audit trail of who made what changes and when

## Usage Examples

### Basic Local Git Repository

```go
gitConfig := &repository.GitConfig{
    RemoteURL: "", // No remote
    Username:  "local-user",
    Token:     "",
    Branch:    "main",
}

repo, err := repository.NewGitRepositoryV2("/path/to/workspace", "My Workspace", gitConfig)
if err != nil {
    log.Fatal(err)
}

// All operations are automatically committed
workspace := domain.NewWorkspace("My Workspace")
err = repo.CreateWorkspace(workspace)
// This creates a commit: "Add workspace: My Workspace"
```

### Remote Git Repository

```go
gitConfig := &repository.GitConfig{
    RemoteURL: "https://github.com/team/chapar-workspace.git",
    Username:  "team-member",
    Token:     "ghp_xxxxxxxxxxxx",
    Branch:    "main",
}

repo, err := repository.NewGitRepositoryV2("/path/to/workspace", "Team Workspace", gitConfig)
if err != nil {
    log.Fatal(err)
}

// Changes are automatically pushed to remote
request := domain.NewHTTPRequest("API Test")
err = repo.CreateRequest(request, nil)
// This creates a commit and pushes to remote
```

### Manual Git Operations

```go
// Get commit history
commits, err := repo.GetCommitHistory()
for _, commit := range commits {
    fmt.Println(commit)
}

// Manual push (usually automatic)
err = repo.PushChanges()

// Manual pull (usually automatic)
err = repo.PullChanges()
```

## Migration from Filesystem

To migrate from the filesystem backend to Git:

1. Enable Git in the application settings
2. Configure your Git credentials and remote URL
3. Restart the application
4. The existing data will be automatically committed to Git

## Best Practices

1. **Use meaningful commit messages**: The system automatically generates descriptive commit messages, but you can customize them if needed.

2. **Regular synchronization**: If using a remote repository, ensure regular pushes and pulls to keep team members synchronized.

3. **Branch management**: Use different branches for different features or environments.

4. **Backup**: While Git provides version control, always maintain backups of your remote repository.

5. **Access control**: Use appropriate Git repository permissions to control who can access your Chapar workspace.

## Troubleshooting

### Authentication Issues
- Ensure your Git credentials are correct
- For GitHub, use personal access tokens instead of passwords
- Check that your token has the necessary permissions

### Merge Conflicts
- Git will handle most conflicts automatically
- For complex conflicts, resolve them using standard Git tools
- The application will show error messages if manual intervention is needed

### Network Issues
- The application will continue to work locally even if remote operations fail
- Failed pushes/pulls will be retried on the next operation
- Check your network connection and remote repository accessibility

## Security Considerations

1. **Token Security**: Store Git tokens securely and never commit them to version control
2. **Repository Access**: Use private repositories for sensitive API configurations
3. **Audit Trail**: Git provides a complete audit trail of all changes
4. **Access Control**: Implement proper Git repository permissions for team collaboration
