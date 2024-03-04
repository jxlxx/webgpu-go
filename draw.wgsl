
struct VertexInput {
  @location(0) pos: vec2<f32>,
  @builtin(instance_index) instance: u32,
};

struct VertexOutput {
  @builtin(position) pos: vec4<f32>,
  @location(0) cell: vec2<f32>, // New line!
};

@group(0) @binding(0) var<uniform> grid: vec2<f32>;

@vertex
fn main_vs(input: VertexInput) -> VertexOutput{

    let i = f32(input.instance);
    
    let cell = vec2<f32>(i % grid.x, floor(i / grid.x));
    let cellOffset = cell / grid * 2.0;
    let gridPos = (input.pos + 1.0) / grid - 1.0 + cellOffset;
    
    var output: VertexOutput;
    output.pos = vec4<f32>(gridPos, 0.0, 1.0);
    output.cell = cell; // New line!
    return output;
}

@fragment
fn main_fs(input: VertexOutput) -> @location(0) vec4<f32> {
    let c = input.cell / grid;
    return vec4<f32>(c, 1.0-c.x, 1.0);
}
