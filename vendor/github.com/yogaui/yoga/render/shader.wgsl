// UI shader: screen-space pixels -> NDC via uniform; three fragment modes:
// flat fill, mono atlas tint, color atlas sample.
//
// textureSample calls must be in uniform control flow (browser WebGPU);
// always sample, then select the result.

struct Uniforms {
    screen : vec4<f32>,
};

@group(0) @binding(0) var<uniform> u : Uniforms;
@group(0) @binding(1) var monoAtlas : texture_2d<f32>;
@group(0) @binding(2) var colorAtlas : texture_2d<f32>;
@group(0) @binding(3) var atlasSampler : sampler;

struct VSOut {
    @builtin(position) clip : vec4<f32>,
    @location(0) uv : vec2<f32>,
    @location(1) color : vec4<f32>,
    @location(2) page : f32,
};

@vertex
fn vs_main(
    @location(0) pos : vec2<f32>,
    @location(1) uv : vec2<f32>,
    @location(2) color : vec4<f32>,
    @location(3) page : f32,
) -> VSOut {
    var out : VSOut;
    let ndcX = pos.x / u.screen.x * 2.0 - 1.0;
    let ndcY = 1.0 - pos.y / u.screen.y * 2.0;
    out.clip = vec4<f32>(ndcX, ndcY, 0.0, 1.0);
    out.uv = uv;
    out.color = color;
    out.page = page;
    return out;
}

@fragment
fn fs_main(in : VSOut) -> @location(0) vec4<f32> {
    // Clamp so flat-fill vertices (uv.x < 0) still sample a defined texel.
    let sampleUV = clamp(in.uv, vec2<f32>(0.0), vec2<f32>(1.0));
    let mono = textureSample(monoAtlas, atlasSampler, sampleUV);
    let colorSamp = textureSample(colorAtlas, atlasSampler, sampleUV);

    if (in.uv.x < 0.0 || in.page < 0.5) {
        return in.color;
    }
    if (in.page < 1.5) {
        return vec4<f32>(in.color.rgb, in.color.a * mono.r);
    }
    return vec4<f32>(colorSamp.rgb, colorSamp.a * in.color.a);
}
