package variables

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
)

// GetVariables returns is a map of variables that can be used in the request body, headers, and query parameters.
func GetVariables() map[string]string {
	return map[string]string{
		"randomUUID4":   uuid.NewString(),
		"timeNow":       time.Now().UTC().Format(time.RFC3339),
		"unixTimestamp": strconv.FormatInt(time.Now().UTC().Unix(), 10),
		"randInt1000":   strconv.Itoa(rand.Intn(1000)),
		"randInt100":    strconv.Itoa(rand.Intn(100)),
		"randInt":       strconv.Itoa(rand.Int()),
		"randFloat":     strconv.FormatFloat(rand.Float64(), 'f', 6, 64),
		"randFloat32":   strconv.FormatFloat(rand.Float64(), 'f', 6, 32),
		"fullDate":      time.Now().UTC().Format(time.RFC1123Z),
		"randBool":      strconv.FormatBool(rand.Intn(2) == 1),
	}
}

// ApplyToEnv apply variables to the environment
func ApplyToEnv(variables map[string]string, env *domain.EnvSpec) {
	if variables == nil {
		variables = GetVariables()
	}

	if env == nil {
		return
	}

	// to through all the variables and replace them in the environment
	for k, v := range variables {
		for i, kv := range env.Values {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				env.Values[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}
	}

	// add env variables to variables
	for _, kv := range env.Values {
		if !kv.Enable {
			continue
		}
		variables[kv.Key] = kv.Value
	}
}

// ApplyToGRPCRequest apply variables to the request
func ApplyToGRPCRequest(variables map[string]string, req *domain.GRPCRequestSpec) {
	if variables == nil {
		variables = GetVariables()
	}

	if req == nil {
		return
	}

	// to through all the variables and replace them in the request
	for k, v := range variables {
		if strings.Contains(req.ServerInfo.Address, "{{"+k+"}}") {
			req.ServerInfo.Address = strings.ReplaceAll(req.ServerInfo.Address, "{{"+k+"}}", v)
		}

		// if value contain the variable in double curly braces then replace it
		if strings.Contains(req.Body, "{{"+k+"}}") {
			req.Body = strings.ReplaceAll(req.Body, "{{"+k+"}}", v)
		}

		for i, kv := range req.Metadata {
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Metadata[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}

		if req.Auth != (domain.Auth{}) {
			ApplyToAuth(variables, &req.Auth)
		}
	}
}

func ApplyToAuth(variables map[string]string, auth *domain.Auth) {
	if variables == nil {
		variables = GetVariables()
	}

	if auth == nil {
		return
	}

	for k, v := range variables {
		if auth.TokenAuth != nil {
			if strings.Contains(auth.TokenAuth.Token, "{{"+k+"}}") {
				auth.TokenAuth.Token = strings.ReplaceAll(auth.TokenAuth.Token, "{{"+k+"}}", v)
			}
		}

		if auth.BasicAuth != nil {
			if strings.Contains(auth.BasicAuth.Username, "{{"+k+"}}") {
				auth.BasicAuth.Username = strings.ReplaceAll(auth.BasicAuth.Username, "{{"+k+"}}", v)
			}

			if strings.Contains(auth.BasicAuth.Password, "{{"+k+"}}") {
				auth.BasicAuth.Password = strings.ReplaceAll(auth.BasicAuth.Password, "{{"+k+"}}", v)
			}
		}

		if auth.APIKeyAuth != nil {
			if strings.Contains(auth.APIKeyAuth.Key, "{{"+k+"}}") {
				auth.APIKeyAuth.Key = strings.ReplaceAll(auth.APIKeyAuth.Key, "{{"+k+"}}", v)
			}

			if strings.Contains(auth.APIKeyAuth.Value, "{{"+k+"}}") {
				auth.APIKeyAuth.Value = strings.ReplaceAll(auth.APIKeyAuth.Value, "{{"+k+"}}", v)
			}
		}
	}
}
