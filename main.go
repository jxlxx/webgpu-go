package main

import (
	"fmt"
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

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	window.MakeContextCurrent()

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
}

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
