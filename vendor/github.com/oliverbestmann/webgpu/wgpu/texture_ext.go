package wgpu

func (g *Texture) AsImageCopy() *TexelCopyTextureInfo {
	return &TexelCopyTextureInfo{
		Texture:  g,
		MipLevel: 0,
		Origin:   Origin3D{},
		Aspect:   TextureAspectAll,
	}
}
