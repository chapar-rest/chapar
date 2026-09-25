// Package langsrv decides which language servers Chapar runs, and when.
//
// Editors fall into three roles, and only the first gets a server by default:
//
//   - Scripts (Python pre/post-request actions) are code people write, so the
//     Python server is on out of the box.
//   - Request bodies (JSON, XML) usually hold {{variable}} templates a strict
//     server flags as errors, so their servers are opt-in.
//   - Generated code in the code dialog is a read-only snippet; its servers
//     (gopls, jdtls, ...) are heavy and opt-in.
//
// Response viewers and plain text never get one. A server starts only when an
// editor of its language is first on screen, is shared by every editor of that
// language, and stops when the last of them closes.
package langsrv

import (
	"slices"

	"github.com/chapar-rest/chapar/internal/domain"
)

// Language is a language Chapar can run a server for.
type Language struct {
	ID      string // config key (domain.LanguageServerConfig.Language)
	LSPID   string // LSP languageId sent to the server
	Name    string
	Ext     string // extension of the editors' virtual documents
	Command string // default server executable
	Args    []string
	Enabled bool   // on by default
	Install string // how to get the default server
	Note    string // shown under the language in settings
	// Installers are the commands that can install the default server, in
	// order of preference; the first whose tool is on PATH is offered.
	Installers []Installer
	URL        string // project page with manual install steps
}

// Languages lists every configurable language, in settings order.
var Languages = []Language{
	{
		ID: "python", LSPID: "python", Name: "Python", Ext: ".py",
		Command: "pyright-langserver", Args: []string{"--stdio"}, Enabled: true,
		Install: "npm install -g pyright  (or: pip install pyright)",
		Note:    "Pre/post-request scripts",
		Installers: []Installer{
			{"npm", []string{"install", "-g", "pyright"}},
			{"brew", []string{"install", "pyright"}},
			{"pip3", []string{"install", "--user", "pyright"}},
		},
		URL: "https://github.com/microsoft/pyright",
	},
	{
		ID: "json", LSPID: "json", Name: "JSON", Ext: ".json",
		Command: "vscode-json-language-server", Args: []string{"--stdio"},
		Install: "npm install -g vscode-langservers-extracted",
		Note:    "Request bodies. Flags {{variable}} templates as errors",
		Installers: []Installer{
			{"npm", []string{"install", "-g", "vscode-langservers-extracted"}},
			{"brew", []string{"install", "vscode-langservers-extracted"}},
		},
		URL: "https://github.com/hrsh7th/vscode-langservers-extracted",
	},
	{
		ID: "xml", LSPID: "xml", Name: "XML", Ext: ".xml",
		Command: "lemminx",
		Install: "install lemminx: https://github.com/eclipse/lemminx",
		Note:    "Request bodies",
		URL:     "https://github.com/eclipse/lemminx",
	},
	{
		ID: "go", LSPID: "go", Name: "Go", Ext: ".go",
		Command: "gopls", Args: []string{"serve"},
		Install: "go install golang.org/x/tools/gopls@latest",
		Note:    "Generated code",
		Installers: []Installer{
			{"go", []string{"install", "golang.org/x/tools/gopls@latest"}},
			{"brew", []string{"install", "gopls"}},
		},
		URL: "https://pkg.go.dev/golang.org/x/tools/gopls",
	},
	{
		ID: "javascript", LSPID: "javascript", Name: "JavaScript", Ext: ".js",
		Command: "typescript-language-server", Args: []string{"--stdio"},
		Install: "npm install -g typescript-language-server typescript",
		Note:    "Generated code (Axios, Node fetch)",
		Installers: []Installer{
			{"npm", []string{"install", "-g", "typescript-language-server", "typescript"}},
			{"brew", []string{"install", "typescript-language-server"}},
		},
		URL: "https://github.com/typescript-language-server/typescript-language-server",
	},
	{
		ID: "java", LSPID: "java", Name: "Java", Ext: ".java",
		Command: "jdtls",
		Install: "brew install jdtls",
		Note:    "Generated code. Heavy: starts a JVM",
		Installers: []Installer{
			{"brew", []string{"install", "jdtls"}},
		},
		URL: "https://github.com/eclipse-jdtls/eclipse.jdt.ls",
	},
	{
		ID: "csharp", LSPID: "csharp", Name: "C#", Ext: ".cs",
		Command: "csharp-ls",
		Install: "dotnet tool install --global csharp-ls",
		Note:    "Generated code (.NET)",
		Installers: []Installer{
			{"dotnet", []string{"tool", "install", "--global", "csharp-ls"}},
		},
		URL: "https://github.com/razzmatazz/csharp-language-server",
	},
	{
		ID: "bash", LSPID: "shellscript", Name: "Bash", Ext: ".sh",
		Command: "bash-language-server", Args: []string{"start"},
		Install: "npm install -g bash-language-server",
		Note:    "Generated code (curl)",
		Installers: []Installer{
			{"npm", []string{"install", "-g", "bash-language-server"}},
			{"brew", []string{"install", "bash-language-server"}},
		},
		URL: "https://github.com/bash-lsp/bash-language-server",
	},
	{
		ID: "ruby", LSPID: "ruby", Name: "Ruby", Ext: ".rb",
		Command: "solargraph", Args: []string{"stdio"},
		Install: "gem install solargraph",
		Note:    "Generated code",
		Installers: []Installer{
			{"gem", []string{"install", "--user-install", "solargraph"}},
		},
		URL: "https://solargraph.org",
	},
}

// ByID returns the language with the given config key.
func ByID(id string) (Language, bool) {
	for _, l := range Languages {
		if l.ID == id {
			return l, true
		}
	}
	return Language{}, false
}

// ByLSPID returns the language with the given LSP languageId.
func ByLSPID(id string) (Language, bool) {
	for _, l := range Languages {
		if l.LSPID == id {
			return l, true
		}
	}
	return Language{}, false
}

// Default returns the built-in configuration for l.
func (l Language) Default() domain.LanguageServerConfig {
	return domain.LanguageServerConfig{
		Language: l.ID,
		Enabled:  l.Enabled,
		Command:  l.Command,
		Args:     slices.Clone(l.Args),
	}
}

// Effective returns the configuration in force for every language, in
// Languages order: the user's entry where there is one, the default otherwise.
func Effective(cfg domain.LanguageServersConfig) []domain.LanguageServerConfig {
	out := make([]domain.LanguageServerConfig, 0, len(Languages))
	for _, l := range Languages {
		if c, ok := cfg.Server(l.ID); ok {
			if c.Command == "" {
				c.Command = l.Command
				c.Args = slices.Clone(l.Args)
			}
			out = append(out, c)
			continue
		}
		out = append(out, l.Default())
	}
	return out
}
