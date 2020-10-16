// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

//go:generate go run generate.go

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const generatedGoFile = "../shaders.go"
const vertexShader = "vertex.glsl"
const fragmentShader = "fragment.glsl"
const glslVersion = `"#version 150"`

func read(filename string) (string, error) {
	vs, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer vs.Close()

	s, err := ioutil.ReadAll(vs)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

func generate() bool {
	// open file
	vs, err := read(vertexShader)
	if err != nil {
		fmt.Printf("error opening vertex shader (%s)", err)
		return false
	}

	fs, err := read(fragmentShader)
	if err != nil {
		fmt.Printf("error opening vertex shader (%s)", err)
		return false
	}

	// create output file (over-writing) if it already exists
	f, err := os.Create(generatedGoFile)
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}
	defer f.Close()

	output := strings.Builder{}
	output.WriteString("// Code generated by hardware/gui/sdlimgui/shaders/generate.go DO NOT EDIT\n")
	output.WriteString("\npackage shaders\n\n")
	output.WriteString(fmt.Sprintf("const Vertex=%s + `\n%s`\n\n", glslVersion, vs))
	output.WriteString(fmt.Sprintf("const Fragment=%s + `\n%s`\n", glslVersion, fs))

	_, err = f.WriteString(output.String())
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}

	fmt.Println("vertex and fragment shaders generated")

	return true
}

func main() {
	if !generate() {
		os.Exit(10)
	}
}
