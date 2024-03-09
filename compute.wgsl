@group(0) @binding(0) var<uniform> grid: vec2<f32>;
@group(0) @binding(1) var<storage> cellStateIn: array<u32>;
@group(0) @binding(2) var<storage, read_write> cellStateOut: array<u32>;

@compute
@workgroup_size(16)
fn main(@builtin(global_invocation_id) cell: vec3<u32>) {
      let activeNeighbors = cellActive(cell.x+u32(1), cell.y+u32(1)) +
                            cellActive(cell.x+u32(1), cell.y) +
                            cellActive(cell.x+u32(1), cell.y-u32(1)) +
                            cellActive(cell.x, cell.y-u32(1) +
                            cellActive(cell.x-u32(1), cell.y-u32(1)) +
                            cellActive(cell.x-u32(1), cell.y)) +
                            cellActive(cell.x-u32(1), cell.y+u32(1)) +
                            cellActive(cell.x, cell.y+u32(1));

      let i = cellIndex(cell.xy);

      // Conway's game of life rules:
      switch activeNeighbors {
        case 2u: {
          cellStateOut[i] = cellStateIn[i];
        }
        case 3u: {
          cellStateOut[i] = u32(1);
        }
        default: {
          cellStateOut[i] = u32(0);
        }
      }
}

fn cellActive(x: u32, y: u32) -> u32 {
  return cellStateIn[cellIndex(vec2(x, y))];
}

fn cellIndex(cell: vec2<u32>) -> u32 {
  return (cell.y % u32(grid.y)) * u32(grid.x) + (cell.x % u32(grid.x));
}

