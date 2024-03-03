# readme

## Getting started

On Linux needed to install:
- xorg-dev
- libgl1-mesa-dev


# Open a Window
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
