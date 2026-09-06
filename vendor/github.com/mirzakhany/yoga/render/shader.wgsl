// UI shader: screen-space pixels -> NDC via uniform; three fragment modes:
// flat fill, mono atlas tint, color atlas sample.

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
    if (in.uv.x < 0.0 || in.page < 0.5) {
        return in.color;
    }
    if (in.page < 1.5) {
        let coverage = textureSample(monoAtlas, atlasSampler, in.uv).r;
        return vec4<f32>(in.color.rgb, in.color.a * coverage);
    }
    let col = textureSample(colorAtlas, atlasSampler, in.uv);
    return vec4<f32>(col.rgb, col.a * in.color.a);
}
