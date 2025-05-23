package domain

import (
	"strings"

	"github.com/google/uuid"
)

type Environment struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       EnvSpec  `yaml:"spec"`
}

type EnvSpec struct {
	Values []KeyValue `yaml:"values"`
}

func (e *EnvSpec) Clone() EnvSpec {
	clone := EnvSpec{
		Values: make([]KeyValue, len(e.Values)),
	}

	for i, v := range e.Values {
		clone.Values[i] = KeyValue{
			ID:     uuid.NewString(),
			Key:    v.Key,
			Value:  v.Value,
			Enable: v.Enable,
		}
	}

	return clone
}

func NewEnvironment(name string) *Environment {
	return &Environment{
		ApiVersion: ApiVersion,
		Kind:       KindEnv,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: EnvSpec{
			Values: make([]KeyValue, 0),
		},
	}
}

func CompareEnvValue(a, b KeyValue) bool {
	// compare length of the values
	if len(a.Key) != len(b.Key) || len(a.Value) != len(b.Value) || len(a.ID) != len(b.ID) {
		return false
	}

	if a.Key != b.Key || a.Value != b.Value || a.Enable != b.Enable || a.ID != b.ID {
		return false
	}

	return true
}

func (e *Environment) Clone() *Environment {
	clone := &Environment{
		ApiVersion: e.ApiVersion,
		Kind:       e.Kind,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: e.MetaData.Name,
		},
		Spec: e.Spec.Clone(),
	}

	return clone
}

func (e *Environment) SetKey(key string, value string) {
	for i, v := range e.Spec.Values {
		if v.Key == key {
			e.Spec.Values[i].Value = value
			return
		}
	}

	e.Spec.Values = append(e.Spec.Values, KeyValue{
		ID:     uuid.NewString(),
		Key:    key,
		Value:  value,
		Enable: true,
	})
}

func (e *Environment) ApplyToGRPCRequest(req *GRPCRequestSpec) {
	if e == nil || req == nil {
		return
	}

	for _, envKv := range e.Spec.Values {
		if strings.Contains(req.ServerInfo.Address, "{{"+envKv.Key+"}}") {
			req.ServerInfo.Address = strings.ReplaceAll(req.ServerInfo.Address, "{{"+envKv.Key+"}}", envKv.Value)
		}

		if strings.Contains(req.Body, "{{"+envKv.Key+"}}") {
			req.Body = strings.ReplaceAll(req.Body, "{{"+envKv.Key+"}}", envKv.Value)
		}

		for i, kv := range req.Metadata {
			if strings.Contains(kv.Value, "{{"+kv.Key+"}}") {
				req.Metadata[i].Value = strings.ReplaceAll(kv.Value, "{{"+kv.Key+"}}", kv.Value)
			}
		}

		if req.Auth != (Auth{}) {
			if req.Auth.APIKeyAuth != nil {
				if strings.Contains(req.Auth.APIKeyAuth.Key, "{{"+envKv.Key+"}}") {
					req.Auth.APIKeyAuth.Key = strings.ReplaceAll(req.Auth.APIKeyAuth.Key, "{{"+envKv.Key+"}}", envKv.Value)
				}
			}

			if req.Auth.BasicAuth != nil {
				if strings.Contains(req.Auth.BasicAuth.Username, "{{"+envKv.Key+"}}") {
					req.Auth.BasicAuth.Username = strings.ReplaceAll(req.Auth.BasicAuth.Username, "{{"+envKv.Key+"}}", envKv.Value)
				}

				if strings.Contains(req.Auth.BasicAuth.Password, "{{"+envKv.Key+"}}") {
					req.Auth.BasicAuth.Password = strings.ReplaceAll(req.Auth.BasicAuth.Password, "{{"+envKv.Key+"}}", envKv.Value)
				}
			}

			if req.Auth.TokenAuth != nil {
				if strings.Contains(req.Auth.TokenAuth.Token, "{{"+envKv.Key+"}}") {
					req.Auth.TokenAuth.Token = strings.ReplaceAll(req.Auth.TokenAuth.Token, "{{"+envKv.Key+"}}", envKv.Value)
				}
			}
		}
	}
}

func (e *Environment) ApplyToHTTPRequest(req *HTTPRequestSpec) {
	if e == nil || req == nil {
		return
	}

	for _, envKv := range e.Spec.Values {
		for i, kv := range req.Request.Headers {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+envKv.Key+"}}") {
				req.Request.Headers[i].Value = strings.ReplaceAll(kv.Value, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		for i, kv := range req.Request.PathParams {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+envKv.Key+"}}") {
				req.Request.PathParams[i].Value = strings.ReplaceAll(kv.Value, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		for i, kv := range req.Request.QueryParams {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+envKv.Key+"}}") {
				req.Request.QueryParams[i].Value = strings.ReplaceAll(kv.Value, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		if strings.Contains(req.URL, "{{"+envKv.Key+"}}") {
			req.URL = strings.ReplaceAll(req.URL, "{{"+envKv.Key+"}}", envKv.Value)
		}

		if strings.Contains(req.Request.Body.Data, "{{"+envKv.Key+"}}") {
			req.Request.Body.Data = strings.ReplaceAll(req.Request.Body.Data, "{{"+envKv.Key+"}}", envKv.Value)
		}

		for i, field := range req.Request.Body.FormData.Fields {
			if field.Type == FormFieldTypeFile {
				continue
			}
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(field.Value, "{{"+envKv.Key+"}}") {
				req.Request.Body.FormData.Fields[i].Value = strings.ReplaceAll(field.Value, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		for i, kv := range req.Request.Body.URLEncoded {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+kv.Key+"}}") {
				req.Request.Body.URLEncoded[i].Value = strings.ReplaceAll(kv.Value, "{{"+kv.Key+"}}", kv.Value)
			}
		}

		if req.Request.Auth != (Auth{}) && req.Request.Auth.TokenAuth != nil {
			if strings.Contains(req.Request.Auth.TokenAuth.Token, "{{"+envKv.Key+"}}") {
				req.Request.Auth.TokenAuth.Token = strings.ReplaceAll(req.Request.Auth.TokenAuth.Token, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		if req.Request.Auth != (Auth{}) && req.Request.Auth.BasicAuth != nil {
			if strings.Contains(req.Request.Auth.BasicAuth.Username, "{{"+envKv.Key+"}}") {
				req.Request.Auth.BasicAuth.Username = strings.ReplaceAll(req.Request.Auth.BasicAuth.Username, "{{"+envKv.Key+"}}", envKv.Value)
			}

			if strings.Contains(req.Request.Auth.BasicAuth.Password, "{{"+envKv.Key+"}}") {
				req.Request.Auth.BasicAuth.Password = strings.ReplaceAll(req.Request.Auth.BasicAuth.Password, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}

		if req.Request.Auth != (Auth{}) && req.Request.Auth.APIKeyAuth != nil {
			if strings.Contains(req.Request.Auth.APIKeyAuth.Key, "{{"+envKv.Key+"}}") {
				req.Request.Auth.APIKeyAuth.Key = strings.ReplaceAll(req.Request.Auth.APIKeyAuth.Key, "{{"+envKv.Key+"}}", envKv.Value)
			}

			if strings.Contains(req.Request.Auth.APIKeyAuth.Value, "{{"+envKv.Key+"}}") {
				req.Request.Auth.APIKeyAuth.Value = strings.ReplaceAll(req.Request.Auth.APIKeyAuth.Value, "{{"+envKv.Key+"}}", envKv.Value)
			}
		}
	}
}

func (e *Environment) GetKeyValues() map[string]interface{} {
	values := make(map[string]interface{})

	for _, kv := range e.Spec.Values {
		values[kv.Key] = kv.Value
	}

	return values
}
