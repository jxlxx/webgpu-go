package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/rajveermalviya/go-webgpu/wgpu"
	wgpuext_glfw "github.com/rajveermalviya/go-webgpu/wgpuext/glfw"
)

const GRID_SIZE = 128 // creates a GRID_SIZE x GRID_SIZE grid

type State struct {
	instance      *wgpu.Instance
	surface       *wgpu.Surface
	swapChain     *wgpu.SwapChain
	device        *wgpu.Device
	queue         *wgpu.Queue
	config        *wgpu.SwapChainDescriptor
	pipeline      *wgpu.RenderPipeline
	vertexBuffer  *wgpu.Buffer
	gridBuffer    *wgpu.Buffer
	vertices      []float32
	grid          []float32
	gridBindGroup *wgpu.BindGroup
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

func InitState(window *glfw.Window) (s *State, err error) {
	s = &State{}

	instance := wgpu.CreateInstance(nil)
	defer instance.Release()

	s.instance = instance

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

	v := [...]float32{
		// X, Y,
		-0.8, -0.8, // Triangle 1
		0.8, -0.8,
		0.8, 0.8,
		-0.8, -0.8, // Triangle 2
		0.8, 0.8,
		-0.8, 0.8,
	}

	s.vertices = v[:]

	vertexBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "Cell Vertices",
		Contents: wgpu.ToBytes(s.vertices[:]),
		Usage:    wgpu.BufferUsage_Vertex | wgpu.BufferUsage_CopyDst,
	})
	if err != nil {
		return s, err
	}
	s.vertexBuffer = vertexBuffer

	s.grid = []float32{GRID_SIZE, GRID_SIZE}
	// for x := range s.grid {
	// s.grid[x] = 1
	// }

	gridBuffer, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "grid",
		Contents: wgpu.ToBytes(s.grid[:][:]),
		Usage:    wgpu.BufferUsage_Uniform | wgpu.BufferUsage_CopyDst,
	})
	if err != nil {
		return s, err
	}
	s.gridBuffer = gridBuffer

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
	s.queue.WriteBuffer(s.gridBuffer, 0, wgpu.ToBytes(s.grid[:]))

	renderPipelineLayout, err := s.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            "Render Pipeline Layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{},
	})
	if err != nil {
		return s, err
	}
	defer renderPipelineLayout.Release()

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

	pipeline, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Render Pipeline",
		// Layout: renderPipelineLayout,
		Vertex: wgpu.VertexState{
			Module:     drawShader,
			EntryPoint: "main_vs",
			Buffers:    bufferLayouts,
		},
		Fragment: &wgpu.FragmentState{
			Module:     drawShader,
			EntryPoint: "main_fs",
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

	bindGroupLayout := s.pipeline.GetBindGroupLayout(0)
	defer bindGroupLayout.Release()

	bindGroup, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: bindGroupLayout,
		Label:  "grid bind group",
		Entries: []wgpu.BindGroupEntry{
			{
				Binding: 0,
				Buffer:  s.gridBuffer,
				Size:    wgpu.WholeSize,
			},
		},
	})
	if err != nil {
		return s, err
	}

	s.gridBindGroup = bindGroup

	return s, err
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
	renderPass := commandEncoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{attachColourToView(nextTexture)},
	})
	defer renderPass.Release()

	renderPass.SetPipeline(s.pipeline)
	renderPass.SetVertexBuffer(0, s.vertexBuffer, 0, wgpu.WholeSize)
	renderPass.SetBindGroup(0, s.gridBindGroup, nil)
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
	if s.vertexBuffer != nil {
		s.vertexBuffer.Release()
		s.vertexBuffer = nil
	}
}
