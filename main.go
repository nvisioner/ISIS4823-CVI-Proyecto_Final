package main

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"github.com/nvisioner/glutils/cam"
	"github.com/nvisioner/glutils/gfx"
	"github.com/nvisioner/glutils/models"
	"github.com/nvisioner/glutils/primitives"
	"github.com/nvisioner/glutils/win"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	width  = 1280
	height = 720
	title  = "Core"
)

var (
	lightPositions = []mgl32.Vec3{
		{0, 1, -2},
		{5, 1, 2},
	}
	lightColors = []mgl32.Vec3{
		{1, 1, 0.7},
		{1, 1, 0.7},
	}
)

func createVAO(vertices, normals, tCoords []float32, indices []uint32) uint32 {

	var VAO uint32
	gl.GenVertexArrays(1, &VAO)
	gl.BindVertexArray(VAO)

	var VBO uint32
	gl.GenBuffers(1, &VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	var NBO uint32
	gl.GenBuffers(1, &NBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, NBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(normals)*4, gl.Ptr(normals), gl.STATIC_DRAW)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	if len(tCoords) > 0 {
		var TBO uint32
		gl.GenBuffers(1, &TBO)
		gl.BindBuffer(gl.ARRAY_BUFFER, TBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(tCoords)*4, gl.Ptr(tCoords), gl.STATIC_DRAW)
		gl.VertexAttribPointer(2, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(2)
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	}

	var EBO uint32
	gl.GenBuffers(1, &EBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	gl.BindVertexArray(0)

	return VAO
}

func pointLightsUL(program *gfx.Program) [][]int32 {
	uniformLocations := [][]int32{}
	for i := 0; i < len(lightPositions); i++ {
		uniformLocations = append(uniformLocations,
			[]int32{program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].position")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].ambient")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].diffuse")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].specular")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].constant")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].linear")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].quadratic")),
				program.GetUniformLocation(fmt.Sprint("pointLights[", i, "].lightColor"))})
	}
	return uniformLocations
}

func programLoop(window *win.Window) error {

	// Shaders and textures
	vS, err := gfx.NewShaderFromFile("shaders/phong_ml.vert", gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fS, err := gfx.NewShaderFromFile("shaders/phong_ml.frag", gl.FRAGMENT_SHADER)
	if err != nil {
		return err
	}

	program, err := gfx.NewProgram(vS, fS)
	if err != nil {
		return err
	}
	defer program.Delete()

	sourceVS, err := gfx.NewShaderFromFile("shaders/source.vert", gl.VERTEX_SHADER)
	if err != nil {
		return err
	}

	sourceFS, err := gfx.NewShaderFromFile("shaders/source.frag", gl.FRAGMENT_SHADER)
	if err != nil {
		return err
	}

	// special shader program so that lights themselves are not affected by lighting
	sourceProgram, err := gfx.NewProgram(sourceVS, sourceFS)
	if err != nil {
		return err
	}
	defer sourceProgram.Delete()

	// Ensure that triangles that are "behind" others do not draw over top of them
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	//gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)

	// Base model
	model := mgl32.Ident4()

	// Uniform
	modelUL := program.GetUniformLocation("model")
	viewUL := program.GetUniformLocation("view")
	projectUL := program.GetUniformLocation("projection")
	objectColorUL := program.GetUniformLocation("objectColor")
	viewPosUL := program.GetUniformLocation("viewPos")
	numLightsUL := program.GetUniformLocation("numLights")
	textureUL := program.GetUniformLocation("texture_diffuse1")
	timeUL := program.GetUniformLocation("time")
	pointLightsUL := pointLightsUL(program)

	sourceModelUL := sourceProgram.GetUniformLocation("model")
	sourceViewUL := sourceProgram.GetUniformLocation("view")
	sourceProjectUL := sourceProgram.GetUniformLocation("projection")
	sourceObjectColorUL := sourceProgram.GetUniformLocation("objectColor")
	//sourceTextureUL := sourceProgram.GetUniformLocation("texSampler")

	// creates camara
	eye := mgl32.Vec3{0, 1.5, 5}
	camera := cam.NewFpsCamera(eye, mgl32.Vec3{0, -1, 0}, 90, 0, window.InputManager())

	// creates perspective
	fov := float32(60.0)
	projection := mgl32.Perspective(mgl32.DegToRad(fov), float32(width)/height, 0.1, 100)

	// Textures
	planTexture, err := gfx.NewTextureFromFile("textures/snow.jpg",
		gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	if err != nil {
		panic(err.Error())
	}

	// Settings
	backgroundColor := mgl32.Vec3{0, 0, 0}
	objectColor := mgl32.Vec3{1.0, 1.0, 1.0}
	polygonMode := false

	if polygonMode {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}

	// Geometry

	xLightSegments, yLighteSegments := 30, 30
	lightVAO := createVAO(primitives.Sphere(xLightSegments, yLighteSegments))

	xPlanSegments, yPlanSegments := 30, 30
	planVAO := createVAO(primitives.Square(xPlanSegments, yPlanSegments, 1))

	// model loading
	houseModel, _ := models.NewModel("./models/", "house.obj", false)
	chairNTableModel, _ := models.NewModel("./models/", "ChairNTable.obj", false)
	bearModel, _ := models.NewModel("./models/", "bear.obj", false)

	// Scene and animation always needs to be after the model and buffers initialization
	animationCtl := gfx.NewAnimationManager()

	animationCtl.Init() // always needs to be before the main loop in order to get correct times

	time := float32(0.0)
	animationCtl.AddAnimation(
		func(t float32) {
			time = t
		}, 45,
	)

	// main loop
	for !window.ShouldClose() {
		window.StartFrame()

		// background color
		gl.ClearColor(backgroundColor.X(), backgroundColor.Y(), backgroundColor.Z(), 1.)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Scene update
		animationCtl.Update()
		camera.Update(window.SinceLastFrame())
		eye = camera.GetPos()

		// You shall draw here
		program.Use()
		gl.Uniform1f(timeUL, time)

		camTransform := camera.GetTransform()
		gl.UniformMatrix4fv(viewUL, 1, false, &camTransform[0])
		gl.UniformMatrix4fv(projectUL, 1, false, &projection[0])

		gl.Uniform3fv(viewPosUL, 1, &eye[0])
		gl.Uniform3f(objectColorUL, objectColor.X(), objectColor.Y(), objectColor.Z())
		gl.Uniform1i(numLightsUL, int32(len(lightPositions)))

		//Lights
		for index, pointLightPosition := range lightPositions {
			gl.Uniform3fv(pointLightsUL[index][0], 1, &pointLightPosition[0])
			gl.Uniform3f(pointLightsUL[index][1], 0.05, 0.05, 0.05)
			gl.Uniform3f(pointLightsUL[index][2], 0.8, 0.8, 0.8)
			gl.Uniform3f(pointLightsUL[index][3], 1.0, 1.0, 1.0)
			gl.Uniform1f(pointLightsUL[index][4], 1.0)
			gl.Uniform1f(pointLightsUL[index][5], 0.09)
			gl.Uniform1f(pointLightsUL[index][6], 0.032)
			gl.Uniform3f(pointLightsUL[index][7], lightColors[index].X(), lightColors[index].Y(), lightColors[index].Z())
		}

		// render models
		modelTransform := model.Mul4(mgl32.Translate3D(0, 0, 0)).Mul4(mgl32.Scale3D(0.7, 0.7, 0.7))
		gl.UniformMatrix4fv(modelUL, 1, false, &modelTransform[0])
		houseModel.Draw(program.Get())
		chairNTableModel.Draw(program.Get())
		bearModel.Draw(program.Get())
		gl.BindVertexArray(0)

		//Plan
		gl.BindVertexArray(planVAO)
		planTexture.Bind(gl.TEXTURE0)
		planTexture.SetUniform(textureUL)
		gl.UniformMatrix4fv(modelUL, 1, false, &model[0])
		gl.DrawElements(gl.TRIANGLES, int32(xPlanSegments*yPlanSegments)*6, gl.UNSIGNED_INT, unsafe.Pointer(nil))
		planTexture.UnBind()
		gl.BindVertexArray(0)

		//Source program
		sourceProgram.Use()
		gl.UniformMatrix4fv(sourceProjectUL, 1, false, &projection[0])
		gl.UniformMatrix4fv(sourceViewUL, 1, false, &camTransform[0])

		//Light objects
		gl.BindVertexArray(lightVAO)
		for i, lp := range lightPositions {
			lightTransform := model
			lightTransform = model.Mul4(mgl32.Translate3D(lp.Elem())).Mul4(mgl32.Scale3D(0.2, 0.2, 0.2))
			gl.Uniform3f(sourceObjectColorUL, lightColors[i].X(), lightColors[i].Y(), lightColors[i].Z())
			gl.UniformMatrix4fv(sourceModelUL, 1, false, &lightTransform[0])
			gl.DrawElements(gl.TRIANGLES, int32(xLightSegments*yLighteSegments)*6, gl.UNSIGNED_INT, unsafe.Pointer(nil))
		}
		gl.BindVertexArray(0)
	}

	return nil
}

func main() {

	runtime.LockOSThread()

	win.InitGlfw(4, 0)

	defer glfw.Terminate()

	window := win.NewWindow(width, height, title)
	gfx.InitGl()
	fmt.Println(gl.MAX_GEOMETRY_OUTPUT_VERTICES)
	err := programLoop(window)
	if err != nil {
		log.Fatal(err)
	}
}
