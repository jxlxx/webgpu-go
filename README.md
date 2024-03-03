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


## Juicy part

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
computePass := commandEncoder.BeginComputePass(nil)
defer computePass.Release()
```
