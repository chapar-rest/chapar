package grpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
)

// exampleTimeout bounds how long creating a collection waits for the first
// example body. Examples need the method descriptors, which may mean asking a
// server that is down; the collection is still made, with empty bodies.
const exampleTimeout = 5 * time.Second

// promptCreateCollection asks for a name, then makes a collection with one
// request per loaded method.
func (c *Container) promptCreateCollection() {
	if c.deps.Dialogs == nil {
		return
	}
	dialogs := c.deps.Dialogs()
	if dialogs == nil {
		return
	}
	dialogs.ShowInputValue("New collection from methods", "Collection name", c.defaultCollectionName(),
		func(name string) {
			name = strings.TrimSpace(name)
			if name == "" {
				name = c.defaultCollectionName()
			}
			c.createCollection(name)
		}, nil)
}

// defaultCollectionName is the service's short name when there is one
// service, else the server address.
func (c *Container) defaultCollectionName() string {
	spec := c.req.Spec.GRPC
	if len(spec.Services) == 1 {
		name := spec.Services[0].Name
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		if name != "" {
			return name
		}
	}
	if spec.ServerInfo.Address != "" {
		return spec.ServerInfo.Address
	}
	return "New Collection"
}

// createCollection builds the collection off the UI thread: filling each
// request with an example body can reach the server.
func (c *Container) createCollection(name string) {
	if c.creatingCollection {
		return
	}
	c.flush()
	c.creatingCollection = true
	src := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		col, err := buildCollection(c.deps, src, env, name)
		c.resultCh <- result{collection: col, collectionErr: err}
		c.deps.WakeNow()
	}()
}

// buildCollection saves a collection named name holding one request per
// method of src. Each request copies src's connection, metadata, auth,
// variables and scripts, selects its method, and gets an example body.
func buildCollection(deps container.Deps, src *domain.Request, env *domain.Environment, name string) (*domain.Collection, error) {
	spec := src.Spec.GRPC
	col := domain.NewCollection(name)
	if err := deps.Repo.CreateCollection(col); err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}
	examples := true
	for _, svc := range spec.Services {
		for _, m := range svc.Methods {
			req := src.Clone()
			req.MetaData.Name = m.Name
			if len(spec.Services) > 1 {
				req.MetaData.Name = shortName(svc.Name) + "." + m.Name
			}
			req.MetaData.Description = ""
			req.CollectionID = col.MetaData.ID
			req.CollectionName = col.MetaData.Name
			req.Spec.GRPC.LasSelectedMethod = m.FullName
			req.Spec.GRPC.Body = "{}"
			if examples {
				body, err := exampleBody(deps, src, env, m.FullName)
				if err != nil {
					// The first failure is taken to hold for every method
					// (the server is down, a proto is missing), so the rest
					// are not waited on.
					examples = false
				} else {
					req.Spec.GRPC.Body = body
				}
			}
			if err := deps.Repo.CreateRequest(req, col); err != nil {
				return col, fmt.Errorf("create request %s: %w", req.MetaData.Name, err)
			}
			col.AddRequest(req)
		}
	}
	return col, nil
}

// exampleBody generates an example message for method, giving up after
// exampleTimeout. The lookup reuses src's ID so the descriptors loaded for it
// are reused rather than fetched again per method.
func exampleBody(deps container.Deps, src *domain.Request, env *domain.Environment, method string) (string, error) {
	req := container.CopyRequest(src)
	req.Spec.GRPC.LasSelectedMethod = method
	type res struct {
		body string
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		body, err := deps.Sender.GRPCExampleBody(req, env)
		ch <- res{body, err}
	}()
	select {
	case r := <-ch:
		return r.body, r.err
	case <-time.After(exampleTimeout):
		return "", fmt.Errorf("timed out generating an example for %s", method)
	}
}

func shortName(service string) string {
	if i := strings.LastIndex(service, "."); i >= 0 {
		return service[i+1:]
	}
	return service
}

// collectionCreated refreshes the sidebar and opens the new collection.
func (c *Container) collectionCreated(col *domain.Collection, err error) {
	c.creatingCollection = false
	if err != nil {
		c.deps.ShowError(err)
	}
	if col == nil {
		return
	}
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved() // reloads the catalog and the trees
	}
	if c.deps.OpenCollection != nil {
		if fresh := c.deps.Catalog.CollectionByID(col.MetaData.ID); fresh != nil {
			col = fresh
		}
		c.deps.OpenCollection(col)
	}
	if err == nil {
		c.deps.Toast(fmt.Sprintf("Created %q with %s", col.MetaData.Name, plural(len(col.Spec.Requests), "request")))
	}
}
