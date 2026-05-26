//go:build js

package wgpu

// ShaderModuleDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpushadermoduledescriptor
type ShaderModuleDescriptor struct {
	Label      string
	WGSLSource *ShaderSourceWGSL
}

func (g ShaderModuleDescriptor) toJS() any {
	return map[string]any{
		"code": g.WGSLSource.Code,
	}
}
