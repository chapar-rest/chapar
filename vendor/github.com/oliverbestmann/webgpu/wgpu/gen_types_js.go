//go:build js

package wgpu

import "syscall/js"

type Buffer struct {
	jsValue js.Value
}

func (g *Buffer) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Buffer) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Buffer) toJS() any {
	return g.jsValue
}

func (g *Buffer) markShared() {}

type CommandEncoder struct {
	jsValue js.Value
}

func (g *CommandEncoder) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *CommandEncoder) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *CommandEncoder) toJS() any {
	return g.jsValue
}

func (g *CommandEncoder) markShared() {}

type ComputePassEncoder struct {
	jsValue js.Value
}

func (g *ComputePassEncoder) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *ComputePassEncoder) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *ComputePassEncoder) toJS() any {
	return g.jsValue
}

func (g *ComputePassEncoder) markShared() {}

type Queue struct {
	jsValue js.Value
}

func (g *Queue) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Queue) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Queue) toJS() any {
	return g.jsValue
}

func (g *Queue) markShared() {}

type RenderPassEncoder struct {
	jsValue js.Value
}

func (g *RenderPassEncoder) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *RenderPassEncoder) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *RenderPassEncoder) toJS() any {
	return g.jsValue
}

func (g *RenderPassEncoder) markShared() {}

type Surface struct {
	jsValue js.Value
}

func (g *Surface) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Surface) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Surface) toJS() any {
	return g.jsValue
}

func (g *Surface) markShared() {}

type Texture struct {
	jsValue js.Value
}

func (g *Texture) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Texture) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Texture) toJS() any {
	return g.jsValue
}

func (g *Texture) markShared() {}

type Adapter struct {
	jsValue js.Value
}

func (g *Adapter) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Adapter) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Adapter) toJS() any {
	return g.jsValue
}

func (g *Adapter) markShared() {}

type BindGroup struct {
	jsValue js.Value
}

func (g *BindGroup) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *BindGroup) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *BindGroup) toJS() any {
	return g.jsValue
}

func (g *BindGroup) markShared() {}

type BindGroupLayout struct {
	jsValue js.Value
}

func (g *BindGroupLayout) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *BindGroupLayout) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *BindGroupLayout) toJS() any {
	return g.jsValue
}

func (g *BindGroupLayout) markShared() {}

type CommandBuffer struct {
	jsValue js.Value
}

func (g *CommandBuffer) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *CommandBuffer) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *CommandBuffer) toJS() any {
	return g.jsValue
}

func (g *CommandBuffer) markShared() {}

type ComputePipeline struct {
	jsValue js.Value
}

func (g *ComputePipeline) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *ComputePipeline) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *ComputePipeline) toJS() any {
	return g.jsValue
}

func (g *ComputePipeline) markShared() {}

type Device struct {
	jsValue js.Value
}

func (g *Device) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Device) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Device) toJS() any {
	return g.jsValue
}

func (g *Device) markShared() {}

type Instance struct {
	jsValue js.Value
}

func (g *Instance) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Instance) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Instance) toJS() any {
	return g.jsValue
}

func (g *Instance) markShared() {}

type PipelineLayout struct {
	jsValue js.Value
}

func (g *PipelineLayout) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *PipelineLayout) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *PipelineLayout) toJS() any {
	return g.jsValue
}

func (g *PipelineLayout) markShared() {}

type QuerySet struct {
	jsValue js.Value
}

func (g *QuerySet) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *QuerySet) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *QuerySet) toJS() any {
	return g.jsValue
}

func (g *QuerySet) markShared() {}

type RenderBundle struct {
	jsValue js.Value
}

func (g *RenderBundle) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *RenderBundle) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *RenderBundle) toJS() any {
	return g.jsValue
}

func (g *RenderBundle) markShared() {}

type RenderBundleEncoder struct {
	jsValue js.Value
}

func (g *RenderBundleEncoder) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *RenderBundleEncoder) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *RenderBundleEncoder) toJS() any {
	return g.jsValue
}

func (g *RenderBundleEncoder) markShared() {}

type RenderPipeline struct {
	jsValue js.Value
}

func (g *RenderPipeline) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *RenderPipeline) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *RenderPipeline) toJS() any {
	return g.jsValue
}

func (g *RenderPipeline) markShared() {}

type Sampler struct {
	jsValue js.Value
}

func (g *Sampler) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *Sampler) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *Sampler) toJS() any {
	return g.jsValue
}

func (g *Sampler) markShared() {}

type ShaderModule struct {
	jsValue js.Value
}

func (g *ShaderModule) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *ShaderModule) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *ShaderModule) toJS() any {
	return g.jsValue
}

func (g *ShaderModule) markShared() {}

type TextureView struct {
	jsValue js.Value
}

func (g *TextureView) Release() {
	// no-op, just here to keep api compatibly with native version
}

func (g *TextureView) IsValid() bool {
	// as long as the instance is reachable it is valid
	return g != nil
}

func (g *TextureView) toJS() any {
	return g.jsValue
}

func (g *TextureView) markShared() {}
