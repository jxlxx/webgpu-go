# readme

## Getting started

On Linux needed to install:
- xorg-dev
- libgl1-mesa-dev


# 1. open a window

- this is taken from the readme in [go-gl/glfw](https://github.com/go-gl/glfw)

- this is using Go bindings for GLFW 3 

- GLFW is a small C library that allows the creation and management of windows
 with OpenGL contexts
- making it also possible to use multiple monitors and video modes. 
- It provides access to input from the keyboard, mouse, and joysticks.
- The API provides a thin, multi-platform abstraction layer, primarily for 
applications whose sole graphics output is through the OpenGL API

```go
package main

import (
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	for !window.ShouldClose() {
		// Do OpenGL stuff.
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
```
## `runtime.LockOSThread()`

- LockOSThread wires the calling goroutine to its current operating system thread. The calling goroutine will always execute in that thread, and no other goroutine will execute in it, until the calling goroutine has made as many calls to UnlockOSThread as to LockOSThread. If the calling goroutine exits without unlocking the thread, the thread will be terminated.
- All init functions are run on the startup thread. Calling LockOSThread from an init function will cause the main function to be invoked on that thread. 
- A goroutine should call LockOSThread before calling OS services or non-Go library functions that depend on per-thread state. 

## `glfw.Init`

- Init initializes the GLFW library. Before most GLFW functions can be used, GLFW must be initialized, and before a program terminates GLFW should be terminated in order to free any resources allocated during or after initialization.
- If this function fails, it calls Terminate before returning. If it succeeds, you should call Terminate before the program exits.
- Additional calls to this function after successful initialization but before termination will succeed but will do nothing. 
- This function may only be called from the main thread. 

## `glfw.Terminate`

- Terminate destroys all remaining windows, frees any allocated resources and sets the library to an uninitialized state. Once this is called, you must again call Init successfully before you will be able to use most GLFW functions.
- If GLFW has been successfully initialized, this function should be called before the program exits. If initialization fails, there is no need to call this function, as it is called by Init before it returns failure.
- This function may only be called from the main thread. 

## `glfw.Createwindow`

```go
func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error)
```
- CreateWindow creates a window and its associated context. Most of the options controlling how the window and its context should be created are specified through Hint.

- Successful creation does not change which context is current. Before you can use the newly created context, you need to make it current using MakeContextCurrent.

- Note that the created window and context may differ from what you requested, as not all parameters and hints are hard constraints. This includes the size of the window, especially for full screen windows.


## OpenGL stuff

- SwapBuffers swaps the front and back buffers of the window. If the swap interval is greater than zero, the GPU driver waits the specified number of screen updates before swapping the buffers. 
- PollEvents processes only those events that have already been received and then returns immediately. Processing events will cause the window and input callbacks associated with those events to be called. Can only be called from the main thread.

## Swapchain

In computer graphics, a swap chain (also swapchain) is a series of virtual 
framebuffers used by the graphics card and graphics API for frame rate 
stabilization, stutter reduction, and several other purposes. 
- Because of these benefits, many graphics APIs require the use of a swap chain. 
- The swap chain usually exists in graphics memory, but it can exist in system 
memory as well. 
- A swap chain with two buffers is a double buffer. 

- In every swap chain there are at least two buffers. 
- The first framebuffer, the screenbuffer, is the buffer that is rendered to 
the output of the video card. 
- The remaining buffers are known as backbuffers. 
- Each time a new frame is displayed, the first backbuffer in the swap chain 
takes the place of the screenbuffer, this is called presentation or swapping 
or flipping.

- A variety of other actions may be taken on the previous screenbuffer and other 
backbuffers (if they exist). 
- The screenbuffer may be simply overwritten or returned to the back of the 
swap chain for further processing. 
- The action taken is decided by the client application and is API dependent. 

# 2. hints and destroy window

```go
glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)
if err != nil {
  panic(err)
}
defer window.Destroy()
```

- WindowHint sets hints for the next call to CreateWindow. The hints, once set, retain their values until changed by a call to WindowHint or DefaultWindowHints, or until the library is terminated with Terminate. 
- Destroy destroys the window


## 3. State, Render, Resize, & Destroy

create the following:

```go

type State struct {
}

func InitState(window *glfw.Window) (s *State, err error) { 
  return nil, nil
}

func (s *State) Resize(width, height int) {

}

func (s *State) Render() error {
  return nil
}

func (s *State) Destroy() {

}
```

update main: 
```go
s, err := InitState(window)
if err != nil {
  panic(err)
}
defer s.Destroy()

window.SetSizeCallback(func(w *glfw.Window, width, height int) {
  s.Resize(width, height)
})

for !window.ShouldClose() {
  glfw.PollEvents()

  if err := s.Render(); err != nil {
  	fmt.Println("error occured while rendering:", err)
  }
}
```


# Juicy part

```go
type State struct {
  surface   *wgpu.Surface
  swapChain *wgpu.SwapChain
  device    *wgpu.Device
  queue     *wgpu.Queue
  config    *wgpu.SwapChainDescriptor
}
```

```go
func InitState(window *glfw.Window) (s *State, err error) {
  s = &State{}

  instance := wgpu.CreateInstance(nil)
  defer instance.Release()

  s.surface = instance.CreateSurface(wgpuext_glfw.GetSurfaceDescriptor(window))

  adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
  	ForceFallbackAdapter: forceFallbackAdapter,
  	CompatibleSurface:    s.surface,
  })
  if err != nil {
  	return s, err
  }
  defer adapter.Release()

  s.device, err = adapter.RequestDevice(nil)
  if err != nil {
  	return s, err
  }
  s.queue = s.device.GetQueue()

  caps := s.surface.GetCapabilities(adapter)

  width, height := window.GetSize()
  s.config = &wgpu.SwapChainDescriptor{
  	Usage:       wgpu.TextureUsage_RenderAttachment,
  	Format:      caps.Formats[0],
  	Width:       uint32(width),
  	Height:      uint32(height),
  	PresentMode: wgpu.PresentMode_Fifo,
  	AlphaMode:   caps.AlphaModes[0],
  }

  s.swapChain, err = s.device.CreateSwapChain(s.surface, s.config)
  if err != nil {
  	return s, err
  }

  return s, nil
}
```

- The instance is the first thing you create when using wgpu. 
Its main purpose is to create Adapters and Surfaces.

- You can think of an adapter as WebGPU's representation of a specific piece of GPU hardware in your device.
- Get the adapter with: `func (p *Instance) RequestAdapter(options *RequestAdapterOptions) (*Adapter, error)`

```go
type RequestAdapterOptions struct {
  CompatibleSurface    *Surface
  PowerPreference      PowerPreference
  ForceFallbackAdapter bool
  BackendType          BackendType
}
```

- The `force_fallback_adapter` forces wgpu to pick an adapter that will work on all hardware. 
This usually means that the rendering backend will use a "software" system instead of hardware such as a GPU.

- The surface is the part of the window that we draw to. 

- The usage field describes how SurfaceTextures will be used. 
- RENDER_ATTACHMENT specifies that the textures will be used to write to the screen 

- The format defines how SurfaceTextures will be stored on the GPU. 

- width and height are the width and the height in pixels of a SurfaceTexture. 
This should usually be the width and the height of the window.

- present_mode determines how to sync the surface with the display. 
-PresentMode_Fifo will cap the display rate at the display's framerate. 
This is essentially VSync. This mode is guaranteed to be supported on all platforms. 
- VSync, short for vertical synchronization, is a graphics technology designed to sync a game’s frame rate with the refresh rate of a gaming monitor.


## Render


- The GetCurrentTextureView function will wait for the surface to provide a new 
TextureView that we will render to.

```go
nextTexture, err := s.swapChain.GetCurrentTextureView()
if err != nil {
  return err
}
defer nextTexture.Release()
```


- We need a command encoder to send intructions to the GPU
- Most modern graphics frameworks expect commands to be stored in a command buffer before being sent to the GPU.

```go
commandEncoder, err := s.device.CreateCommandEncoder(nil)
if err != nil {
  return err
}
defer commandEncoder.Release()
```
We need to use the encoder to create a RenderPass. The RenderPass has all the methods for the actual drawing

```go 
renderPass := commandEncoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
  ColorAttachments: []wgpu.RenderPassColorAttachment{
    {
      View:    nextTexture,
      LoadOp:  wgpu.LoadOp_Clear,
      StoreOp: wgpu.StoreOp_Store,
      ClearValue: wgpu.Color{
        R: 0.0,
        G: 0.01,
        B: 0.05,
        A: 1.0,
      },
    },
  },
})
```

Also, modify the loop 

```go
for !window.ShouldClose() {
  glfw.PollEvents()

  if err := s.Render(); err != nil {
    fmt.Println("error occured while rendering:", err)
     switch {
     case errors.Is(err, errors.New("Surface timed out")):
     case errors.Is(err, errors.New("Surface is outdated")):
     case errors.Is(err, errors.New("Surface was lost")):
     default:
     // do nothing (for now)
     }
  }
}
```



# Triangles, Shaders & Pipelines

Your GPU really only deals with a few different types of shapes 
(or primitives as they're referred to by WebGPU): points, lines, and triangles

GPUs work almost exclusively with triangles because triangles have a lot of nice 
mathematical properties that make them easy to process in a predictable and 
efficient way

GPUs rely on small programs called vertex shaders to perform whatever math is 
necessary to transform the vertices into clip space, 
as well as any other calculations needed to draw the vertices.

From there, the GPU takes all the triangles made up by these transformed 
vertices and determines which pixels on the screen are needed to draw them. 
Then it runs another small program you write called a fragment shader that 
calculates what color each pixel should be.

- https://kenny-designs.github.io/zim-websites/opengl/Shaders_and_the_Rendering_Pipeline.html

Shaders are a part of the rendering pipeline that we can make changes to. 
The rendering pipeline is a series of stages that take place in order to 
render an image to the screen. Four of these stages are programmable via shaders.

There are 9 parts but some people may split the stages into more or less categories. This following list will do:
- Vertex Specification
- Vertex Shader (programmable)
- Tessellation (programmable)
- Geometry Shader (programmable)
- Vertex Post-Processing
  - This is the end of all the vertex operations
- Primitive Assembly
  - Handles groups of vertices
- Rasterization
  - The conversion to fragments
- Fragment Shader (programmable)
- Per-Sample Operations
  - Operations performed on the fragments before being rendered to the screen

first, we'll define vertices in go:

```go
vertexData := [...]float32{
  // X, Y,
  -0.8, -0.8, // Triangle 1 
  0.8, -0.8,
  0.8, 0.8,
  -0.8, -0.8, // Triangle 2 
  0.8, 0.8,
  -0.8, 0.8,
}
```

The first thing to notice is that you give the buffer a label. 
Every single WebGPU object you create can be given an optional label, 
and you definitely want to do so! The label is any string you want, as 
long as it helps you identify what the object is. If you run into any 
problems, those labels are used in the error messages WebGPU produces 
to help you understand what went wrong.

Next, give a size for the buffer in bytes. You need a buffer with 48 bytes, 
which you determine by multiplying the size of a 32-bit float ( 4 bytes) 
by the number of floats in your vertices array (12).

Finally, you need to specify the usage of the buffer. 
This is one or more of the GPUBufferUsage flags, with multiple flags 
being combined with the | ( bitwise OR) operator. In this case, you 
specify that you want the buffer to be used for vertex data (GPUBufferUsage.VERTEX) 
and that you also want to be able to copy data into it (GPUBufferUsage.COPY_DST).

```go
vertexBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
  Label:    "Cell Vertices",
  Contents: wgpu.ToBytes(vertexData[:]),
  Usage:    wgpu.BufferUsage_Vertex | wgpu.BufferUsage_CopyDst,
})
if err != nil {
  return s, err
}
defer vertexBuffer.Release()
```

Shaders are mini-programs that you send to the GPU to perform operations 
on your data. There are three main types of shaders: 
vertex, fragment, and compute.

Shaders in WebGPU are written in a shading language called WGSL 
(WebGPU Shading Language). 

WGSL is, syntactically, a bit like Rust, with features aimed at 
making common types of GPU work (like vector and matrix math) 
easier and faster.

A vertex shader must return at least the final
position of the vertex being processed in clip space.
This is always given as a 4-dimensional vector. 

```glsl
@vertex
fn vertexMain() -> @builtin(pos) vec4<f32> {
}  
```

What you want instead is to make use of the data from the buffer that you created, 
and you do that by declaring an argument for your function with a @location() a
ttribute and type that match what you described in the vertexBufferLayout. 

You specified a shaderLocation of 0, so in your WGSL code, mark the argument 
with @location(0). You also defined the format as a float32x2, which is a 2D 
vector, so in WGSL your argument is a vec2f. You can name it whatever you like, 
but since these represent your vertex positions, a name like pos seems natural.

```glsl
@vertex
fn vertexMain(@location(0) pos: vec2f) -> @builtin(position) vec4<f32> {
  return vec4<f32>(0, 0, 0, 1);
}
```

Next up is the fragment shader. Fragment shaders operate in a very similar 
way to vertex shaders, but rather than being invoked for every vertex, 
they're invoked for every pixel being drawn.

Fragment shaders are always called after vertex shaders. The GPU takes 
the output of the vertex shaders and triangulates it, creating triangles 
out of sets of three points. It then rasterizes each of those triangles by 
figuring out which pixels of the output color attachments are included in 
that triangle, and then calls the fragment shader once for each of those 
pixels. The fragment shader returns a color, typically calculated from 
values sent to it from the vertex shader and assets like textures, which 
the GPU writes to the color attachment.

Final draw.wgsl:

```glsl
@vertex
fn vertexMain(@location(0) pos: vec2<f32>) -> 
    @builtin(position) vec4<f32>{
    return vec4<f32>(pos, 0.0, 1.0);
}

@fragment
fn fragmentMain() -> @location(0) vec4<f32> {
    return vec4<f32>(1.0, 1.0, 1.0, 1.0);
}
```

Next create the shader:

```go
drawShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
  Label: "draw.wgsl",
  WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{
  	Code: draw,
  },
})
if err != nil {
  return s, err
}
defer drawShader.Release()
```
Defining the vertex layout:

```go
bufferLayouts := []wgpu.VertexBufferLayout{
  {
  	ArrayStride: 8,
  	StepMode:    wgpu.VertexStepMode_Vertex,
  	Attributes: []wgpu.VertexAttribute{
  		{
  			Format:         wgpu.VertexFormat_Float32x2,
  			Offset:         0,
  			ShaderLocation: 0,
  		},
  	},
  },
}
```

The first thing you give is the arrayStride. This is the number of bytes the 
GPU needs to skip forward in the buffer when it's looking for the next vertex.

Next is the attributes property, which is an array. 
Attributes are the individual pieces of information encoded into each vertex.
We just have position for now, but more advanced applications could have more
(for example velocity).

In your single attribute, you first define the format of the data. 
This comes from a list of GPUVertexFormat types that describe each 
type of vertex data that the GPU can understand.

If the vertex data was instead made up of four 16-bit unsigned integers 
each, you'd use uint16x4, etc.

Next, the offset describes how many bytes into the vertex this particular 
attribute starts. You really only have to worry about this if your buffer 
has more than one attribute in it.

Finally, you have the shaderLocation. This is an arbitrary number between
0 and 15 and must be unique for every attribute that you define.

Now create the render pipeline.

The render pipeline is the most complex object in the entire API, 
but ost of the values you can pass to it are optional, and you 
only need to provide a few to start.

```go
s.pipeline, err = s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
  Vertex: wgpu.VertexState{
  	Module:     drawShader,
  	EntryPoint: "vertexMain",
  	Buffers:    bufferLayouts,
  },
  Fragment: &wgpu.FragmentState{
  	Module:     drawShader,
  	EntryPoint: "fragmentMain",
  	Targets: []wgpu.ColorTargetState{
  		{
  			Format:    s.config.Format,
        Blend:     &wgpu.BlendState_Replace,
  			WriteMask: wgpu.ColorWriteMask_All,
  		},
  	},
  },
  Primitive: wgpu.PrimitiveState{
  	Topology:  wgpu.PrimitiveTopology_TriangleList,
  	FrontFace: wgpu.FrontFace_CCW,
  },
  Multisample: wgpu.MultisampleState{
  	Count:                  1,
  	Mask:                   0xFFFFFFFF,
  	AlphaToCoverageEnabled: false,
  },
})
if err != nil {
  return s, err
}
```

Every pipeline needs a layout that describes what types of inputs 
(other than vertex buffers) the pipeline needs, which we don't have. 

So we don't have to set it and the pipeline builds its own layout from the shaders.

Next, you have to provide details about the vertex stage. The module is the
GPUShaderModule that contains your vertex shader, and the entryPoint gives 
the name of the function in the shader code that is called for every vertex 
invocation.

And now render a square:

```go
defer renderPass.Release()

renderPass.SetPipeline(s.pipeline)
renderPass.SetVertexBuffer(0, s.vertexBuffer, 0, wgpu.WholeSize)
renderPass.Draw(6, 1, 0, 0)

renderPass.End()
```

# Creating a Grid

set a grid size:

```go
const GRID_SIZE = 4;
```

First, you need to communicate the grid size you've chosen to the shader, 
since it uses that to change how things display. You could just hard-code 
the size into the shader, but then that means that any time you want to 
change the grid size you have to re-create the shader and render pipeline, 
which is expensive. A better way is to provide the grid size to the shader 
as uniforms.

You learned earlier that a different value from the vertex buffer is passed 
to every invocation of a vertex shader. A uniform is a value from a buffer 
that is the same for every invocation. They're useful for communicating 
values that are common for a piece of geometry (like its position), a full 
frame of animation (like the current time), or even the entire lifespan of 
the app (like a user preference).

```go
gridData := [GRID_SIZE][GRID_SIZE]uint32{}
s.grid = gridData

gridBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
  Label:    "Grid",
  Contents: wgpu.ToBytes(gridData[:]),
  Usage:    wgpu.BufferUsage_Uniform | wgpu.BufferUsage_CopyDst,
})
if err != nil {
  return s, err
}
s.gridBuffer = gridBuffer
```


```glsl
@group(0) @binding(0) var<uniform> grid: vec2f;

@vertex
fn vertexMain(@location(0) pos: vec2f) ->
  @builtin(position) vec4f {
  return vec4f(pos / grid, 0, 1);
}

// ...fragmentMain is unchanged 
```

Declaring the uniform in the shader doesn't connect it with the buffer that you 
created, though. In order to do that, you need to create and set a bind group.


# Appendix 

## clip space

- [clip space](https://www.scratchapixel.com/lessons/3d-basic-rendering/perspective-and-orthographic-projection-matrix/projection-matrix-GPU-rendering-pipeline-clipping.html)
- [clip coordinates](https://en.wikipedia.org/wiki/Clip_coordinates)
