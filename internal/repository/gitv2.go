package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type GitRepositoryV2 struct {
	repo         *git.Repository
	workTree     *git.Worktree
	dataDir      string
	workspaceName string
	remoteURL    string
	username     string
	token        string
	
	// entities is a map to hold loaded entities so git can track name changes
	entities map[string]string
}

type GitConfig struct {
	RemoteURL string
	Username  string
	Token     string
	Branch    string
}

func NewGitRepositoryV2(dataDir, workspaceName string, gitConfig *GitConfig) (*GitRepositoryV2, error) {
	gr := &GitRepositoryV2{
		dataDir:       dataDir,
		workspaceName: workspaceName,
		entities:      make(map[string]string),
	}

	if gitConfig != nil {
		gr.remoteURL = gitConfig.RemoteURL
		gr.username = gitConfig.Username
		gr.token = gitConfig.Token
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create directory %s: %w", dataDir, err)
	}

	// Initialize or open Git repository
	var err error
	gr.repo, err = gr.initOrOpenRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	gr.workTree, err = gr.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Ensure the workspace directory exists
	workspacePath := filepath.Join(dataDir, workspaceName)
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			return nil, fmt.Errorf("could not create workspace directory %s: %w", workspacePath, err)
		}
	}

	return gr, nil
}

func (g *GitRepositoryV2) initOrOpenRepo() (*git.Repository, error) {
	// Check if repository already exists
	if repo, err := git.PlainOpen(g.dataDir); err == nil {
		return repo, nil
	}

	// Initialize new repository
	repo, err := git.PlainInit(g.dataDir, false)
	if err != nil {
		return nil, err
	}

	// If remote URL is provided, add remote and configure
	if g.remoteURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{g.remoteURL},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create remote: %w", err)
		}
	}

	return repo, nil
}

func (g *GitRepositoryV2) SetActiveWorkspace(workspaceName string) {
	g.workspaceName = workspaceName
}

func (g *GitRepositoryV2) commitChanges(message string) error {
	// Add all changes
	_, err := g.workTree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Check if there are changes to commit
	status, err := g.workTree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		return nil // No changes to commit
	}

	// Commit changes
	_, err = g.workTree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  g.username,
			Email: g.username + "@chapar.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

func (g *GitRepositoryV2) pushChanges() error {
	if g.remoteURL == "" || g.token == "" {
		return nil // No remote configured, skip push
	}

	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	err := g.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

func (g *GitRepositoryV2) pullChanges() error {
	if g.remoteURL == "" || g.token == "" {
		return nil // No remote configured, skip pull
	}

	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	err := g.workTree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	return nil
}

// LoadProtoFiles loads proto files from the Git repository
func (g *GitRepositoryV2) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	// First pull latest changes
	if err := g.pullChanges(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	dir, err := g.EntityPath(domain.KindProtoFile)
	if err != nil {
		return nil, err
	}

	return loadList[domain.ProtoFile](dir, func(n *domain.ProtoFile) {
		g.entities[n.ID()] = n.GetName()
	})
}

func (g *GitRepositoryV2) CreateProtoFile(protoFile *domain.ProtoFile) error {
	g.entities[protoFile.ID()] = protoFile.GetName()
	
	if err := g.writeProtoFile(protoFile, false); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Add proto file: %s", protoFile.GetName()))
}

func (g *GitRepositoryV2) UpdateProtoFile(protoFile *domain.ProtoFile) error {
	oldEntityName, ok := g.entities[protoFile.ID()]
	if !ok {
		return fmt.Errorf("proto file with ID %s not found", protoFile.ID())
	}

	// did the proto file change its name?
	if oldEntityName != protoFile.GetName() {
		// as name has changed, we need to rename the file
		path, err := g.EntityPath(protoFile.GetKind())
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, protoFile.GetName()+".yaml"), protoFile.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			n := g.ensureUniqueName(path, protoFile.GetName(), ".yaml")
			protoFile.SetName(n)
		}

		if err := g.renameEntity(path, oldEntityName+".yaml", protoFile.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename proto file with ID %s: %v", protoFile.ID(), err)
		}

		// Update the name in the entities map
		g.entities[protoFile.ID()] = protoFile.GetName()
	}

	if err := g.writeProtoFile(protoFile, true); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Update proto file: %s", protoFile.GetName()))
}

func (g *GitRepositoryV2) DeleteProtoFile(protoFile *domain.ProtoFile) error {
	path, err := g.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}

	if err := g.deleteEntity(path, protoFile); err != nil {
		return err
	}

	// Remove the proto file from the entities map
	delete(g.entities, protoFile.ID())
	
	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Delete proto file: %s", protoFile.GetName()))
}

// LoadRequests loads standalone requests from the Git repository
func (g *GitRepositoryV2) LoadRequests() ([]*domain.Request, error) {
	// First pull latest changes
	if err := g.pullChanges(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	dir, err := g.EntityPath(domain.KindRequest)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Request](dir, func(n *domain.Request) {
		g.entities[n.ID()] = n.GetName()
	})
}

func (g *GitRepositoryV2) CreateRequest(request *domain.Request, collection *domain.Collection) error {
	// add the request to the entities map but break the pointer to avoid sharing the same object
	g.entities[request.ID()] = request.GetName()
	
	if err := g.writeStandaloneRequest(request, collection, false); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Add request: %s", request.GetName()))
}

func (g *GitRepositoryV2) UpdateRequest(request *domain.Request, collection *domain.Collection) error {
	oldEntityName, ok := g.entities[request.ID()]
	if !ok {
		return fmt.Errorf("request with ID %s not found", request.ID())
	}

	// did the request change its name?
	if oldEntityName != request.GetName() {
		kind := domain.KindRequest
		if collection != nil {
			kind = domain.KindCollection
		}
		// as name has changed, we need to rename the file
		path, err := g.EntityPath(kind)
		if err != nil {
			return err
		}

		if collection != nil {
			// if request is part of a collection, we need to add the collection name to the path
			path = filepath.Join(path, collection.GetName())
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, request.GetName()+".yaml"), request.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			request.SetName(g.ensureUniqueName(path, request.GetName(), ".yaml"))
		}

		if err := g.renameEntity(path, oldEntityName+".yaml", request.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename request with ID %s: %v", request.ID(), err)
		}

		// Update the name in the entities map
		g.entities[request.ID()] = request.GetName()
	}

	if err := g.writeStandaloneRequest(request, collection, true); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Update request: %s", request.GetName()))
}

func (g *GitRepositoryV2) DeleteRequest(request *domain.Request, collection *domain.Collection) error {
	kind := domain.KindRequest
	if collection != nil {
		kind = domain.KindCollection
	}

	dir, err := g.EntityPath(kind)
	if err != nil {
		return err
	}

	// If the request is part of a collection, we need to delete it from the collection directory
	if collection != nil {
		dir = filepath.Join(dir, collection.GetName())
	}

	if err := g.deleteEntity(dir, request); err != nil {
		return err
	}

	// Remove the request from the entities map
	delete(g.entities, request.ID())
	
	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Delete request: %s", request.GetName()))
}

func (g *GitRepositoryV2) LoadCollections() ([]*domain.Collection, error) {
	// First pull latest changes
	if err := g.pullChanges(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	path, err := g.EntityPath(domain.KindCollection)
	if err != nil {
		return nil, err
	}

	// Load all collection directories
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection directory: %w", err)
	}

	collections := make([]*domain.Collection, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue // Skip non-directory entries
		}

		// Each collection directory should have a "_collection.yaml" file
		collectionFile := filepath.Join(path, dir.Name(), "_collection.yaml")
		if _, err := os.Stat(collectionFile); os.IsNotExist(err) {
			continue // Skip if the collection file does not exist
		}

		// Load the collection from the YAML file
		collection, err := LoadFromYaml[domain.Collection](collectionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load collection %s: %w", dir.Name(), err)
		}

		requests, err := g.loadCollectionRequests(filepath.Join(path, dir.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to load requests for collection %s: %w", dir.Name(), err)
		}
		collection.Spec.Requests = requests

		collections = append(collections, collection)
		g.entities[collection.ID()] = collection.GetName()
	}

	return collections, nil
}

func (g *GitRepositoryV2) loadCollectionRequests(path string) ([]*domain.Request, error) {
	return loadList[domain.Request](path, func(n *domain.Request) {
		// set request default values
		n.SetDefaultValues()
		g.entities[n.ID()] = n.GetName()
	})
}

func (g *GitRepositoryV2) CreateCollection(collection *domain.Collection) error {
	g.entities[collection.ID()] = collection.GetName()
	
	if err := g.writeCollection(collection, false); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Add collection: %s", collection.GetName()))
}

func (g *GitRepositoryV2) UpdateCollection(collection *domain.Collection) error {
	// if the collection already exists, we can just update it otherwise, it means the name has changed
	path, err := g.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	// Check if the collection name has changed
	oldEntityName, ok := g.entities[collection.ID()]
	if ok && oldEntityName != collection.GetName() {
		potentialExistingCollectionPath := filepath.Join(path, collection.GetName(), "_collection.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(potentialExistingCollectionPath, collection.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			collection.SetName(g.ensureUniqueName(path, collection.GetName(), ""))
		}

		// Rename the collection directory if the name has changed
		if err := g.renameEntity(path, oldEntityName, collection.GetName()); err != nil {
			return fmt.Errorf("cannot rename collection with ID %s: %v", collection.ID(), err)
		}

		// Update the name in the entities map
		g.entities[collection.ID()] = collection.GetName()
	}

	collectionPath := filepath.Join(path, collection.GetName())
	// if collection already exists, we can just update it
	if _, err := os.Stat(filepath.Join(collectionPath, "_collection.yaml")); err == nil {
		if err := g.writeMetadataFile(collectionPath, "_collection", collection); err != nil {
			return err
		}
	} else {
		if err := g.writeCollection(collection, true); err != nil {
			return err
		}
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Update collection: %s", collection.GetName()))
}

func (g *GitRepositoryV2) DeleteCollection(collection *domain.Collection) error {
	path, err := g.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	// Delete the collection directory
	if err := g.deleteEntity(path, collection); err != nil {
		return err
	}

	// Remove the collection from the entities map
	delete(g.entities, collection.ID())
	
	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Delete collection: %s", collection.GetName()))
}

func (g *GitRepositoryV2) LoadEnvironments() ([]*domain.Environment, error) {
	// First pull latest changes
	if err := g.pullChanges(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	path, err := g.EntityPath(domain.KindEnv)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Environment](path, func(n *domain.Environment) {
		g.entities[n.ID()] = n.GetName()
	})
}

func (g *GitRepositoryV2) CreateEnvironment(environment *domain.Environment) error {
	g.entities[environment.ID()] = environment.GetName()
	
	if err := g.writeEnvironmentFile(environment, false); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Add environment: %s", environment.GetName()))
}

func (g *GitRepositoryV2) UpdateEnvironment(environment *domain.Environment) error {
	oldEntityName, ok := g.entities[environment.ID()]
	if !ok {
		return fmt.Errorf("environment with ID %s not found", environment.ID())
	}

	// did the environment change its name?
	if oldEntityName != environment.GetName() {
		// as name has changed, we need to rename the file
		path, err := g.EntityPath(domain.KindEnv)
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, environment.GetName()+".yaml"), environment.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			environment.SetName(g.ensureUniqueName(path, environment.GetName(), ".yaml"))
		}

		if err := g.renameEntity(path, oldEntityName+".yaml", environment.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename environment with ID %s: %v", environment.ID(), err)
		}

		// Update the name in the entities map
		g.entities[environment.ID()] = environment.GetName()
	}

	if err := g.writeEnvironmentFile(environment, true); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Update environment: %s", environment.GetName()))
}

func (g *GitRepositoryV2) DeleteEnvironment(environment *domain.Environment) error {
	path, err := g.EntityPath(environment.GetKind())
	if err != nil {
		return err
	}

	if err := g.deleteEntity(path, environment); err != nil {
		return err
	}

	// Remove the environment from the entities map
	delete(g.entities, environment.ID())
	
	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Delete environment: %s", environment.GetName()))
}

func (g *GitRepositoryV2) LoadWorkspaces() ([]*domain.Workspace, error) {
	// First pull latest changes
	if err := g.pullChanges(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Workspaces are stored in the dataDir directly
	path := g.dataDir

	// Load all workspace directories
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	workspaces := make([]*domain.Workspace, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue // Skip non-directory entries
		}

		// Skip .git directory
		if dir.Name() == ".git" {
			continue
		}

		// Each workspace directory should have a "_workspace.yaml" file
		workspaceFile := filepath.Join(path, dir.Name(), "_workspace.yaml")
		if _, err := os.Stat(workspaceFile); os.IsNotExist(err) {
			continue // Skip if the workspace file does not exist
		}

		// Load the workspace from the YAML file
		workspace, err := LoadFromYaml[domain.Workspace](workspaceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load workspace %s: %w", dir.Name(), err)
		}

		workspaces = append(workspaces, workspace)
		g.entities[workspace.ID()] = workspace.GetName()
	}

	return workspaces, nil
}

// CreateWorkspace creates a new workspace and writes it to the Git repository
func (g *GitRepositoryV2) CreateWorkspace(workspace *domain.Workspace) error {
	g.entities[workspace.ID()] = workspace.GetName()
	
	if err := g.writeWorkspace(workspace, false); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Add workspace: %s", workspace.GetName()))
}

// UpdateWorkspace updates an existing workspace and writes it to the Git repository
func (g *GitRepositoryV2) UpdateWorkspace(workspace *domain.Workspace) error {
	oldEntityName, ok := g.entities[workspace.ID()]
	if !ok {
		return fmt.Errorf("workspace with ID %s not found", workspace.ID())
	}

	// did the workspace change its name?
	if oldEntityName != workspace.GetName() {
		potentialExistingWorkspacePath := filepath.Join(g.dataDir, workspace.GetName(), "_workspace.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(potentialExistingWorkspacePath, workspace.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			workspace.SetName(g.ensureUniqueName(g.dataDir, workspace.GetName(), ""))
		}

		// as name has changed, we need to rename the file
		if err := g.renameEntity(g.dataDir, oldEntityName, workspace.GetName()); err != nil {
			return fmt.Errorf("cannot rename workspace with ID %s: %v", workspace.ID(), err)
		}

		// Update the name in the entities map
		g.entities[workspace.ID()] = workspace.GetName()
	}

	if err := g.writeWorkspace(workspace, true); err != nil {
		return err
	}

	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Update workspace: %s", workspace.GetName()))
}

func (g *GitRepositoryV2) DeleteWorkspace(workspace *domain.Workspace) error {
	// Delete the workspace directory
	if err := g.deleteEntity(g.dataDir, workspace); err != nil {
		return err
	}

	// Remove the workspace from the entities map
	delete(g.entities, workspace.ID())
	
	// Commit the changes
	return g.commitChanges(fmt.Sprintf("Delete workspace: %s", workspace.GetName()))
}

// Additional methods for Git-specific operations
func (g *GitRepositoryV2) PushChanges() error {
	return g.pushChanges()
}

func (g *GitRepositoryV2) PullChanges() error {
	return g.pullChanges()
}

func (g *GitRepositoryV2) GetCommitHistory() ([]string, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	cIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	var commits []string
	err = cIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, fmt.Sprintf("%s - %s", c.Hash.String()[:7], c.Message))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

// Helper methods (reused from FilesystemV2)
func (g *GitRepositoryV2) writeWorkspace(workspace *domain.Workspace, override bool) error {
	path, err := g.EntityPath(domain.KindWorkspace)
	if err != nil {
		return err
	}

	if !override {
		// Ensure the workspace name is unique
		uniqueName := g.ensureUniqueName(path, workspace.GetName(), "")
		workspace.SetName(uniqueName)
	}

	workspaceDir := filepath.Join(path, workspace.GetName())
	// Ensure the workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		// Create the workspace directory if it does not exist
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			return fmt.Errorf("failed to create collection directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check collection directory: %w", err)
	}

	// Update the collection metadata file
	if err := g.writeMetadataFile(workspaceDir, "_workspace", workspace); err != nil {
		return err
	}

	g.entities[workspace.ID()] = workspace.GetName()
	return nil
}

func (g *GitRepositoryV2) writeCollection(collection *domain.Collection, override bool) error {
	path, err := g.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	if !override {
		// Ensure the collection name is unique
		uniqueName := g.ensureUniqueName(path, collection.GetName(), "")
		collection.SetName(uniqueName)
	}

	collectionDir := filepath.Join(path, collection.GetName())
	// Ensure the collection directory exists
	if _, err := os.Stat(collectionDir); os.IsNotExist(err) {
		// Create the collection directory if it does not exist
		if err := os.MkdirAll(collectionDir, 0755); err != nil {
			return fmt.Errorf("failed to create collection directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check collection directory: %w", err)
	}

	// Update the collection metadata file
	return g.writeMetadataFile(collectionDir, "_collection", collection)
}

func (g *GitRepositoryV2) writeMetadataFile(path, name string, e Entity) error {
	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s", e.GetName())
	}

	filePath := filepath.Join(path, name+".yaml")
	return os.WriteFile(filePath, data, 0644)
}

func (g *GitRepositoryV2) writeStandaloneRequest(request *domain.Request, collection *domain.Collection, override bool) error {
	kind := domain.KindRequest
	if collection != nil {
		kind = domain.KindCollection
	}

	path, err := g.EntityPath(kind)
	if err != nil {
		return err
	}

	if collection != nil {
		// if the request is part of a collection, we need to add the collection name to the path
		path = filepath.Join(path, collection.GetName())
	}

	return g.writeFile(path, request, override)
}

func (g *GitRepositoryV2) writeProtoFile(protoFile *domain.ProtoFile, override bool) error {
	path, err := g.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}
	return g.writeFile(path, protoFile, override)
}

func (g *GitRepositoryV2) writeEnvironmentFile(environment *domain.Environment, override bool) error {
	path, err := g.EntityPath(domain.KindEnv)
	if err != nil {
		return err
	}
	return g.writeFile(path, environment, override)
}

func (g *GitRepositoryV2) deleteEntity(path string, e Entity) error {
	// if the entity is a workspace or a collection, we need to delete the entire directory
	if e.GetKind() == domain.KindWorkspace || e.GetKind() == domain.KindCollection {
		dir := filepath.Join(path, e.GetName())
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to delete entity directory: %w", err)
		}
		return nil
	} else {
		// For other entities, we delete the specific file
		filePath := filepath.Join(path, e.GetName()+".yaml")
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to delete entity: %w", err)
		}
	}

	return nil
}

// writeFile, writes the protofile, request and environment files to the filesystem.
func (g *GitRepositoryV2) writeFile(path string, e Entity, override bool) error {
	if !override {
		uniqueName := g.ensureUniqueName(path, e.GetName(), ".yaml")
		// Set the unique name to the entity
		e.SetName(uniqueName)
	}

	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s", e.GetName())
	}

	filePath := filepath.Join(path, e.GetName()+".yaml")
	return os.WriteFile(filePath, data, 0644)
}

func (g *GitRepositoryV2) renameEntity(path string, oldName, newName string) error {
	oldFilePath := filepath.Join(path, oldName)
	newFilePath := filepath.Join(path, newName)

	// Check if the old file exists
	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", oldFilePath)
	}

	// Rename the entity
	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to rename file from %s to %s: %w", oldFilePath, newFilePath, err)
	}

	return nil
}

func (g *GitRepositoryV2) ensureUniqueName(path, name, extension string) string {
	filePath := filepath.Join(path, name+extension)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return name
	}

	// If the file already exists, append a suffix to make it unique
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s_%d", name, i)
		newFilePath := filepath.Join(path, newName+extension)
		if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
			return newName
		}
	}
}

func (g *GitRepositoryV2) EntityPath(kind string) (string, error) {
	var path string
	switch kind {
	case domain.KindProtoFile:
		path = filepath.Join(g.dataDir, g.workspaceName, "protofiles")
	case domain.KindCollection:
		path = filepath.Join(g.dataDir, g.workspaceName, "collections")
	case domain.KindEnv:
		path = filepath.Join(g.dataDir, g.workspaceName, "envs")
	case domain.KindRequest:
		path = filepath.Join(g.dataDir, g.workspaceName, "requests")
	default:
		// workspace and old config files are living in the dataDir directly
		path = g.dataDir
	}

	// Ensure the directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	return path, nil
}

// GetLegacyConfig gets the legacy config from the Git repository
func (g *GitRepositoryV2) GetLegacyConfig() (*domain.Config, error) {
	filePath := filepath.Join(g.dataDir, "config.yaml")

	// if config file does not exist, create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		config := domain.NewConfig()
		if err := SaveToYaml(filePath, config); err != nil {
			return nil, err
		}

		return config, nil
	}

	return LoadFromYaml[domain.Config](filePath)
}

// ReadLegacyPreferences reads the preferences
func (g *GitRepositoryV2) ReadLegacyPreferences() (*domain.Preferences, error) {
	pdir := filepath.Join(g.dataDir, g.workspaceName, "preferences")
	filePath := filepath.Join(pdir, "preferences.yaml")
	return LoadFromYaml[domain.Preferences](filePath)
}
