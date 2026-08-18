//go:build !nogpu

package render

import (
	_ "embed"
	"fmt"
	"strings"
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
)

//go:embed shader.wgsl
var shaderSrc string

var vertexBufferLayout = wgpu.VertexBufferLayout{
	ArrayStride: uint64(unsafe.Sizeof(Vertex{})),
	StepMode:    wgpu.VertexStepModeVertex,
	Attributes: []wgpu.VertexAttribute{
		{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
		{Format: wgpu.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1},
		{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2},
		{Format: wgpu.VertexFormatFloat32, Offset: 32, ShaderLocation: 3},
	},
}

// Renderer owns the GPU device, surface, pipeline and growable buffers.
type Renderer struct {
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	surface  *wgpu.Surface
	device   *wgpu.Device
	queue    *wgpu.Queue
	config   *wgpu.SurfaceConfiguration
	pipeline *wgpu.RenderPipeline

	uniformBuf *wgpu.Buffer
	monoTex    *wgpu.Texture
	monoView   *wgpu.TextureView
	colorTex   *wgpu.Texture
	colorView  *wgpu.TextureView
	sampler    *wgpu.Sampler
	bindGroup  *wgpu.BindGroup

	vertexBuf *wgpu.Buffer
	indexBuf  *wgpu.Buffer
	vcap      int
	icap      int

	scaleX, scaleY float32
	ClearColor     Color
}

func dpiScale(fb, logical int) float32 {
	if logical <= 0 {
		return 1
	}
	return float32(fb) / float32(logical)
}

func NewRenderer(sd *wgpu.SurfaceDescriptor, fbW, fbH, logicalW, logicalH int, atlas *FontAtlas) (r *Renderer, err error) {
	defer func() {
		if err != nil {
			r.Destroy()
			r = nil
		}
	}()
	r = &Renderer{
		ClearColor: RGBA8(24, 24, 29, 255),
		scaleX:     dpiScale(fbW, logicalW),
		scaleY:     dpiScale(fbH, logicalH),
	}
	r.instance = wgpu.CreateInstance(nil)
	r.surface = r.instance.CreateSurface(sd)
	r.adapter, err = r.instance.RequestAdapter(&wgpu.RequestAdapterOptions{CompatibleSurface: r.surface})
	if err != nil {
		return r, err
	}
	r.device, err = r.adapter.RequestDevice(nil)
	if err != nil {
		return r, err
	}
	r.queue = r.device.GetQueue()
	caps := r.surface.GetCapabilities(r.adapter)
	r.config = &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      pickFormat(caps.Formats),
		Width:       uint32(fbW),
		Height:      uint32(fbH),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
	}
	r.surface.Configure(r.adapter, r.device, r.config)
	if err = r.createPipeline(); err != nil {
		return r, err
	}
	if err = r.uploadAtlases(atlas); err != nil {
		return r, err
	}
	if err = r.createUniformAndBindGroup(logicalW, logicalH); err != nil {
		return r, err
	}
	return r, nil
}

func (r *Renderer) createPipeline() error {
	shader, err := r.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "ui.wgsl",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shaderSrc},
	})
	if err != nil {
		return err
	}
	defer shader.Release()
	r.pipeline, err = r.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "ui-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{vertexBufferLayout},
		},
		Primitive: wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList, FrontFace: wgpu.FrontFaceCCW, CullMode: wgpu.CullModeNone},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module: shader, EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{{Format: r.config.Format, Blend: &wgpu.BlendStateAlphaBlending, WriteMask: wgpu.ColorWriteMaskAll}},
		},
	})
	return err
}

func (r *Renderer) uploadAtlases(atlas *FontAtlas) error {
	if err := r.uploadMono(atlas); err != nil {
		return err
	}
	if err := r.uploadColor(atlas); err != nil {
		return err
	}
	var err error
	r.sampler, err = r.device.CreateSampler(&wgpu.SamplerDescriptor{
		Label: "atlas-sampler", AddressModeU: wgpu.AddressModeClampToEdge, AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge, MagFilter: wgpu.FilterModeLinear, MinFilter: wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeNearest, LodMaxClamp: 1, MaxAnisotropy: 1,
	})
	return err
}

func (r *Renderer) uploadMono(atlas *FontAtlas) error {
	w, h := atlas.MonoSize()
	extent := wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1}
	tex, err := r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label: "mono-atlas", Size: extent, MipLevelCount: 1, SampleCount: 1, Dimension: wgpu.TextureDimension2D,
		Format: wgpu.TextureFormatR8Unorm, Usage: wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if err != nil {
		return err
	}
	view, err := tex.CreateView(nil)
	if err != nil {
		tex.Release()
		return err
	}
	r.queue.WriteTexture(tex.AsImageCopy(), atlas.MonoPixels(), &wgpu.TextureDataLayout{BytesPerRow: uint32(w), RowsPerImage: uint32(h)}, &extent)
	if r.monoView != nil {
		r.monoView.Release()
	}
	if r.monoTex != nil {
		r.monoTex.Release()
	}
	r.monoTex, r.monoView = tex, view
	return nil
}

func (r *Renderer) uploadColor(atlas *FontAtlas) error {
	w, h := atlas.ColorSize()
	extent := wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1}
	tex, err := r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label: "color-atlas", Size: extent, MipLevelCount: 1, SampleCount: 1, Dimension: wgpu.TextureDimension2D,
		Format: wgpu.TextureFormatRGBA8Unorm, Usage: wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if err != nil {
		return err
	}
	view, err := tex.CreateView(nil)
	if err != nil {
		tex.Release()
		return err
	}
	r.queue.WriteTexture(tex.AsImageCopy(), atlas.ColorPixels(), &wgpu.TextureDataLayout{BytesPerRow: uint32(w * 4), RowsPerImage: uint32(h)}, &extent)
	if r.colorView != nil {
		r.colorView.Release()
	}
	if r.colorTex != nil {
		r.colorTex.Release()
	}
	r.colorTex, r.colorView = tex, view
	return nil
}

func (r *Renderer) UpdateAtlas(atlas *FontAtlas) error {
	if err := r.uploadAtlases(atlas); err != nil {
		return err
	}
	layout := r.pipeline.GetBindGroupLayout(0)
	defer layout.Release()
	bg, err := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label: "ui-bind-group", Layout: layout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.uniformBuf, Size: wgpu.WholeSize},
			{Binding: 1, TextureView: r.monoView},
			{Binding: 2, TextureView: r.colorView},
			{Binding: 3, Sampler: r.sampler},
		},
	})
	if err != nil {
		return err
	}
	if r.bindGroup != nil {
		r.bindGroup.Release()
	}
	r.bindGroup = bg
	atlas.ClearFullRebuild()
	return nil
}

// UpdateAtlasRegion uploads a dirty sub-rectangle.
func (r *Renderer) UpdateAtlasRegion(d DirtyRect) error {
	var tex *wgpu.Texture
	var bpr uint32
	if d.Page == PageMono {
		tex = r.monoTex
		bpr = uint32(d.W)
	} else {
		tex = r.colorTex
		bpr = uint32(d.W * 4)
	}
	if tex == nil {
		return nil
	}
	extent := wgpu.Extent3D{Width: uint32(d.W), Height: uint32(d.H), DepthOrArrayLayers: 1}
	origin := wgpu.Origin3D{X: uint32(d.X), Y: uint32(d.Y), Z: 0}
	r.queue.WriteTexture(&wgpu.ImageCopyTexture{Texture: tex, Origin: origin}, d.Pix, &wgpu.TextureDataLayout{BytesPerRow: bpr, RowsPerImage: uint32(d.H)}, &extent)
	return nil
}

func (r *Renderer) createUniformAndBindGroup(width, height int) error {
	u := [4]float32{float32(width), float32(height), 0, 0}
	var err error
	r.uniformBuf, err = r.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label: "uniforms", Contents: wgpu.ToBytes(u[:]), Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return err
	}
	layout := r.pipeline.GetBindGroupLayout(0)
	defer layout.Release()
	r.bindGroup, err = r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label: "ui-bind-group", Layout: layout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.uniformBuf, Size: wgpu.WholeSize},
			{Binding: 1, TextureView: r.monoView},
			{Binding: 2, TextureView: r.colorView},
			{Binding: 3, Sampler: r.sampler},
		},
	})
	return err
}

func (r *Renderer) Resize(fbW, fbH, logicalW, logicalH int) {
	if fbW <= 0 || fbH <= 0 {
		return
	}
	r.config.Width = uint32(fbW)
	r.config.Height = uint32(fbH)
	r.surface.Configure(r.adapter, r.device, r.config)
	r.scaleX = dpiScale(fbW, logicalW)
	r.scaleY = dpiScale(fbH, logicalH)
	u := [4]float32{float32(logicalW), float32(logicalH), 0, 0}
	r.queue.WriteBuffer(r.uniformBuf, 0, wgpu.ToBytes(u[:]))
}

func (r *Renderer) ensureCapacity(nVerts, nIndices int) error {
	if nVerts > r.vcap {
		if r.vertexBuf != nil {
			r.vertexBuf.Release()
		}
		newCap := grow(r.vcap, nVerts)
		buf, err := r.device.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "vertices", Size: uint64(newCap) * uint64(unsafe.Sizeof(Vertex{})), Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return err
		}
		r.vertexBuf, r.vcap = buf, newCap
	}
	if nIndices > r.icap {
		if r.indexBuf != nil {
			r.indexBuf.Release()
		}
		newCap := grow(r.icap, nIndices)
		buf, err := r.device.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "indices", Size: uint64(newCap) * 4, Usage: wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return err
		}
		r.indexBuf, r.icap = buf, newCap
	}
	return nil
}

func pickFormat(formats []wgpu.TextureFormat) wgpu.TextureFormat {
	for _, f := range formats {
		if !strings.HasSuffix(strings.ToLower(f.String()), "srgb") {
			return f
		}
	}
	return formats[0]
}

func grow(cur, need int) int {
	n := cur
	if n == 0 {
		n = 1024
	}
	for n < need {
		n *= 2
	}
	return n
}

func (r *Renderer) Render(dl *DrawList) error {
	surfaceTex, err := r.surface.GetCurrentTexture()
	if err != nil {
		return err
	}
	view, err := surfaceTex.CreateView(nil)
	if err != nil {
		return err
	}
	defer view.Release()
	encoder, err := r.device.CreateCommandEncoder(nil)
	if err != nil {
		return err
	}
	defer encoder.Release()
	pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View: view, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore,
			ClearValue: wgpu.Color{R: float64(r.ClearColor.R), G: float64(r.ClearColor.G), B: float64(r.ClearColor.B), A: float64(r.ClearColor.A)},
		}},
	})
	if n := len(dl.Indices); n > 0 {
		if err := r.ensureCapacity(len(dl.Vertices), n); err != nil {
			pass.End()
			pass.Release()
			return err
		}
		r.queue.WriteBuffer(r.vertexBuf, 0, wgpu.ToBytes(dl.Vertices))
		r.queue.WriteBuffer(r.indexBuf, 0, wgpu.ToBytes(dl.Indices))
		pass.SetPipeline(r.pipeline)
		pass.SetBindGroup(0, r.bindGroup, nil)
		pass.SetVertexBuffer(0, r.vertexBuf, 0, wgpu.WholeSize)
		pass.SetIndexBuffer(r.indexBuf, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
		if len(dl.Commands) == 0 {
			pass.DrawIndexed(uint32(n), 1, 0, 0, 0)
		} else {
			for _, cmd := range dl.Commands {
				if cmd.IndexCount == 0 {
					continue
				}
				r.setScissor(pass, cmd.Clip)
				pass.DrawIndexed(uint32(cmd.IndexCount), 1, uint32(cmd.IndexStart), 0, 0)
			}
		}
	}
	pass.End()
	pass.Release()
	cmd, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmd.Release()
	r.queue.Submit(cmd)
	r.surface.Present()
	return nil
}

func (r *Renderer) setScissor(pass *wgpu.RenderPassEncoder, clip Rect) {
	fbW, fbH := r.config.Width, r.config.Height
	if clip.W < 0 || clip.H < 0 {
		pass.SetScissorRect(0, 0, fbW, fbH)
		return
	}
	x0 := clampU32(clip.X*r.scaleX, fbW)
	y0 := clampU32(clip.Y*r.scaleY, fbH)
	x1 := clampU32((clip.X+clip.W)*r.scaleX, fbW)
	y1 := clampU32((clip.Y+clip.H)*r.scaleY, fbH)
	pass.SetScissorRect(x0, y0, x1-x0, y1-y0)
}

func clampU32(v float32, hi uint32) uint32 {
	if v <= 0 {
		return 0
	}
	u := uint32(v + 0.5)
	if u > hi {
		return hi
	}
	return u
}

func (r *Renderer) Destroy() {
	if r == nil {
		return
	}
	if r.indexBuf != nil {
		r.indexBuf.Release()
	}
	if r.vertexBuf != nil {
		r.vertexBuf.Release()
	}
	if r.bindGroup != nil {
		r.bindGroup.Release()
	}
	if r.sampler != nil {
		r.sampler.Release()
	}
	if r.monoView != nil {
		r.monoView.Release()
	}
	if r.monoTex != nil {
		r.monoTex.Release()
	}
	if r.colorView != nil {
		r.colorView.Release()
	}
	if r.colorTex != nil {
		r.colorTex.Release()
	}
	if r.uniformBuf != nil {
		r.uniformBuf.Release()
	}
	if r.pipeline != nil {
		r.pipeline.Release()
	}
	if r.queue != nil {
		r.queue.Release()
	}
	if r.device != nil {
		r.device.Release()
	}
	if r.adapter != nil {
		r.adapter.Release()
	}
	if r.surface != nil {
		r.surface.Release()
	}
	if r.instance != nil {
		r.instance.Release()
	}
}

var _ = func() bool {
	if unsafe.Sizeof(Vertex{}) != 36 {
		panic(fmt.Sprintf("Vertex must be 36 bytes, got %d", unsafe.Sizeof(Vertex{})))
	}
	return true
}()
