package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/rajveermalviya/go-webgpu/wgpu"
	wgpuext_glfw "github.com/rajveermalviya/go-webgpu/wgpuext/glfw"
)

const GRID_SIZE = 128 // creates a GRID_SIZE x GRID_SIZE grid

type State struct {
	window    *glfw.Window
	instance  *wgpu.Instance
	adapter   *wgpu.Adapter
	device    *wgpu.Device
	surface   *wgpu.Surface
	queue     *wgpu.Queue
	swapChain *wgpu.SwapChain
	config    *wgpu.SwapChainDescriptor

	pipeline           *wgpu.RenderPipeline
	simulationPipeline *wgpu.ComputePipeline

	vertexBuffer   *wgpu.Buffer
	gridBuffer     *wgpu.Buffer
	vertices       []float32
	grid           []float32
	gridBindGroups []*wgpu.BindGroup
	cellStates     [][]uint32
	steps          int
}

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

	s, err := InitState(window)
	if err != nil {
		panic(err)
	}
	defer s.Destroy()

	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		s.Resize(width, height)
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Print resource usage on pressing 'R'
		if key == glfw.KeyR && (action == glfw.Press || action == glfw.Repeat) {
			report := s.instance.GenerateReport()
			buf, _ := json.MarshalIndent(report, "", "  ")
			fmt.Print(string(buf))
		}
	})

	for !window.ShouldClose() {
		time.Sleep(100 * time.Millisecond)
		glfw.PollEvents()

		if err := s.Render(); err != nil {
			fmt.Println("error occured while rendering:", err)
			switch {
			case errors.Is(err, errors.New("Surface timed out")):
			case errors.Is(err, errors.New("Surface is outdated")):
			case errors.Is(err, errors.New("Surface was lost")):
			default:
				panic(err)
			}
		}
	}
}

var forceFallbackAdapter = os.Getenv("WGPU_FORCE_FALLBACK_ADAPTER") == "1"

//go:embed draw.wgsl
var draw string

//go:embed compute.wgsl
var compute string

func (s *State) setSurface() {
	instance := wgpu.CreateInstance(nil)
	s.instance = instance
	s.surface = instance.CreateSurface(wgpuext_glfw.GetSurfaceDescriptor(s.window))
}

func (s *State) setDevice() {
	adapter, err := s.instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		ForceFallbackAdapter: forceFallbackAdapter,
		CompatibleSurface:    s.surface,
	})
	if err != nil {
		log.Fatalln(err)
	}
	s.adapter = adapter
	s.device, err = adapter.RequestDevice(nil)
	if err != nil {
		log.Fatalln(err)
	}
	s.queue = s.device.GetQueue()
}

func (s *State) setSwapChain() {
	caps := s.surface.GetCapabilities(s.adapter)
	width, height := s.window.GetSize()

	s.config = &wgpu.SwapChainDescriptor{
		Usage:       wgpu.TextureUsage_RenderAttachment,
		Format:      caps.Formats[0],
		Width:       uint32(width),
		Height:      uint32(height),
		PresentMode: wgpu.PresentMode_Fifo,
		AlphaMode:   caps.AlphaModes[0],
	}

	sc, err := s.device.CreateSwapChain(s.surface, s.config)
	if err != nil {
		log.Fatalln(err)
	}
	s.swapChain = sc
}

func (s *State) initVertexBuffer() {
	s.vertices = []float32{
		// X, Y,
		-0.8, -0.8, // Triangle 1
		0.8, -0.8,
		0.8, 0.8,
		-0.8, -0.8, // Triangle 2
		0.8, 0.8,
		-0.8, 0.8,
	}
	vertexBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "Tile Vertices",
		Contents: wgpu.ToBytes(s.vertices[:]),
		Usage:    wgpu.BufferUsage_Vertex | wgpu.BufferUsage_CopyDst,
	})
	if err != nil {
		log.Fatalln(err)
	}
	s.vertexBuffer = vertexBuffer
	if err := s.queue.WriteBuffer(s.vertexBuffer, 0, wgpu.ToBytes(s.vertices[:])); err != nil {
		log.Fatalln(err)
	}
}

func (s *State) initGridBuffer() {
	s.grid = []float32{GRID_SIZE, GRID_SIZE}
	gridBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "grid",
		Contents: wgpu.ToBytes(s.grid[:]),
		Usage:    wgpu.BufferUsage_Uniform | wgpu.BufferUsage_CopyDst,
	})
	if err != nil {
		log.Fatalln(err)
	}
	s.gridBuffer = gridBuffer
	if err := s.queue.WriteBuffer(s.gridBuffer, 0, wgpu.ToBytes(s.grid[:])); err != nil {
		log.Fatalln(err)
	}
}

func (s *State) createShader(label, code string) *wgpu.ShaderModule {
	shader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: label,
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{
			Code: code,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	return shader
}

func InitState(window *glfw.Window) (s *State, err error) {
	defer func() {
		if err != nil {
			s.Destroy()
			s = nil
		}
	}()
	s = &State{
		window: window,
	}
	s.setSurface()
	s.setDevice()
	s.setSwapChain()
	s.initVertexBuffer()
	s.initGridBuffer()

	drawShader := s.createShader("render shader", draw)
	defer drawShader.Release()

	computeShader := s.createShader("compute shader", compute)
	defer computeShader.Release()

	vertexBufferLayout := []wgpu.VertexBufferLayout{
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

	bindGroupLayout, err := s.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "bind group layouts",
		Entries: []wgpu.BindGroupLayoutEntry{
			{
				Binding:    0,
				Visibility: wgpu.ShaderStage_Vertex | wgpu.ShaderStage_Compute | wgpu.ShaderStage_Fragment,
				Buffer: wgpu.BufferBindingLayout{
					Type: wgpu.BufferBindingType_Uniform,
				},
			},
			{
				Binding:    1,
				Visibility: wgpu.ShaderStage_Vertex | wgpu.ShaderStage_Compute,
				Buffer: wgpu.BufferBindingLayout{
					Type: wgpu.BufferBindingType_ReadOnlyStorage,
				},
			},
			{
				Binding:    2,
				Visibility: wgpu.ShaderStage_Compute,
				Buffer: wgpu.BufferBindingLayout{
					Type: wgpu.BufferBindingType_Storage,
				},
			},
		},
	})
	if err != nil {
		return s, err
	}

	renderPipelineLayout, err := s.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label: "Render Pipeline Layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{
			bindGroupLayout,
		},
	})
	if err != nil {
		return s, err
	}
	// defer renderPipelineLayout.Release()

	pipeline, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  "Render Pipeline",
		Layout: renderPipelineLayout,
		Vertex: wgpu.VertexState{
			Module:     drawShader,
			EntryPoint: "main_vs",
			Buffers:    vertexBufferLayout,
		},
		Fragment: &wgpu.FragmentState{
			Module:     drawShader,
			EntryPoint: "main_fs",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    s.config.Format,
					Blend:     nil,
					WriteMask: wgpu.ColorWriteMask_All,
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopology_TriangleList,
			FrontFace: wgpu.FrontFace_CCW,
			CullMode:  wgpu.CullMode_Back,
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

	s.pipeline = pipeline

	s.cellStates = [][]uint32{
		make([]uint32, GRID_SIZE*GRID_SIZE),
		make([]uint32, GRID_SIZE*GRID_SIZE),
	}

	for i := range s.cellStates[0] {
		r := rand.Float32()
		if r > 0.7 {
			s.cellStates[0][i] = 1
			s.cellStates[1][i] = 1
		}
	}

	cellStateStorage := []*wgpu.Buffer{
		s.storageBuffer(wgpu.ToBytes(s.cellStates[0])),
		s.storageBuffer(wgpu.ToBytes(s.cellStates[1])),
	}

	s.gridBindGroups = []*wgpu.BindGroup{
		s.bindGroup("cell renderer A", bindGroupLayout, s.gridBuffer, cellStateStorage[0], cellStateStorage[1]),
		s.bindGroup("cell renderer B", bindGroupLayout, s.gridBuffer, cellStateStorage[1], cellStateStorage[0]),
	}

	computePipeline, err := s.device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Label:  "compute",
		Layout: renderPipelineLayout,
		Compute: wgpu.ProgrammableStageDescriptor{
			Module:     computeShader,
			EntryPoint: "main",
		},
	})
	if err != nil {
		return s, err
	}
	s.simulationPipeline = computePipeline
	return s, err
}

func (s *State) bindGroup(label string, l *wgpu.BindGroupLayout, x, y, w *wgpu.Buffer) *wgpu.BindGroup {
	b, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: l,
		Label:  label,
		Entries: []wgpu.BindGroupEntry{
			{
				Binding: 0,
				Buffer:  x,
				Size:    wgpu.WholeSize,
			},
			{
				Binding: 1,
				Buffer:  y,
				Size:    wgpu.WholeSize,
			},
			{
				Binding: 2,
				Buffer:  w,
				Size:    wgpu.WholeSize,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return b
}

func (s *State) storageBuffer(content []byte) *wgpu.Buffer {
	b, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "cells",
		Contents: content,
		Usage:    wgpu.BufferUsage_Storage | wgpu.BufferUsage_CopyDst,
	})
	if err != nil {
		panic(err)
	}
	return b
}

func (s *State) Resize(width, height int) {
	if width > 0 && height > 0 {
		s.config.Width = uint32(width)
		s.config.Height = uint32(height)

		if s.swapChain != nil {
			s.swapChain.Release()
		}
		var err error
		s.swapChain, err = s.device.CreateSwapChain(s.surface, s.config)
		if err != nil {
			panic(err)
		}
	}
}

func attachColourToView(view *wgpu.TextureView) wgpu.RenderPassColorAttachment {
	return wgpu.RenderPassColorAttachment{
		View:    view,
		LoadOp:  wgpu.LoadOp_Clear,
		StoreOp: wgpu.StoreOp_Store,
		ClearValue: wgpu.Color{
			R: 0.0,
			G: 0.01,
			B: 0.05,
			A: 1.0,
		}}
}

func (s *State) Render() error {
	nextTexture, err := s.swapChain.GetCurrentTextureView()
	if err != nil {
		return err
	}
	defer nextTexture.Release()
	commandEncoder, err := s.device.CreateCommandEncoder(nil)
	if err != nil {
		return err
	}
	defer commandEncoder.Release()

	computePass := commandEncoder.BeginComputePass(nil)
	defer computePass.Release()

	computePass.SetPipeline(s.simulationPipeline)
	computePass.SetBindGroup(0, s.gridBindGroups[s.steps%2], nil)
	computePass.DispatchWorkgroups(GRID_SIZE, GRID_SIZE, 1)
	computePass.End()

	s.steps += 1

	renderPass := commandEncoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{attachColourToView(nextTexture)},
	})
	defer renderPass.Release()

	renderPass.SetPipeline(s.pipeline)
	renderPass.SetBindGroup(0, s.gridBindGroups[s.steps%2], nil)
	renderPass.SetVertexBuffer(0, s.vertexBuffer, 0, wgpu.WholeSize)
	renderPass.Draw(6, GRID_SIZE*GRID_SIZE, 0, 0)
	renderPass.End()

	cmdBuffer, err := commandEncoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmdBuffer.Release()

	s.queue.Submit(cmdBuffer)
	s.swapChain.Present()

	return nil
}

func (s *State) Destroy() {
	if s.swapChain != nil {
		s.swapChain.Release()
		s.swapChain = nil
	}
	if s.instance != nil {
		s.instance.Release()
		s.instance = nil
	}
	if s.config != nil {
		s.config = nil
	}
	if s.queue != nil {
		s.queue.Release()
		s.queue = nil
	}
	if s.device != nil {
		s.device.Release()
		s.device = nil
	}
	if s.surface != nil {
		s.surface.Release()
		s.surface = nil
	}
	if s.adapter != nil {
		s.adapter.Release()
		s.adapter = nil
	}
	if s.vertexBuffer != nil {
		s.vertexBuffer.Release()
		s.vertexBuffer = nil
	}
	if s.gridBuffer != nil {
		s.gridBuffer.Release()
		s.gridBuffer = nil
	}
	if s.pipeline != nil {
		s.pipeline.Release()
		s.pipeline = nil
	}
	if s.simulationPipeline != nil {
		s.simulationPipeline.Release()
		s.simulationPipeline = nil
	}
	for _, bg := range s.gridBindGroups {
		if bg != nil {
			bg.Release()
		}
	}
}
