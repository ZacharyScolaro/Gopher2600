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
	"unicode"
)

const generatedShadersFile = "../shaders.go"
const generatedConstsFile = "../constants.go"
const vertexShader = "vertex.vert"
const fragmentShader = "fragment.frag"

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

func generate() (rb bool) {
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

	// shaders

	shaders, err := os.Create(generatedShadersFile)
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}
	defer func() {
		err := shaders.Close()
		if err != nil {
			fmt.Printf("error during file close: %s\n", err)
			rb = false
		}
	}()

	output := strings.Builder{}
	output.WriteString("// Code generated by hardware/gui/sdlimgui/shaders/generate.go DO NOT EDIT\n")
	output.WriteString("\npackage shaders\n\n")
	output.WriteString(fmt.Sprintf("const Vertex = `%s`\n\n", vs))
	output.WriteString(fmt.Sprintf("const Fragment = `%s`\n", fs))

	_, err = shaders.WriteString(output.String())
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}

	// constsants

	consts, err := os.Create(generatedConstsFile)
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}
	defer func() {
		err := consts.Close()
		if err != nil {
			fmt.Printf("error during file close: %s\n", err)
			rb = false
		}
	}()

	output.Reset()
	output.WriteString("// Code generated by hardware/gui/sdlimgui/shaders/generate.go DO NOT EDIT\n")
	output.WriteString("\npackage shaders\n\n")

	_, err = consts.WriteString(output.String())
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		return false
	}

	// walk through fragment shader looking for constants
	for _, s := range strings.Split(fs, "\n") {
		var t string
		var l string
		var v int

		n, err := fmt.Sscanf(s, "const %s %s = %d", &t, &l, &v)

		if err != nil {
			switch err.Error() {
			case "input does not match format":
			case "unexpected EOF":
			default:
				fmt.Printf("error during instruction table generation: %s\n", err)
				return false
			}
		}

		// convert if enought conversions have taken place and if first
		// character of label is uppercase
		if n == 3 && unicode.IsUpper(rune(l[0])) {
			// convert GLSL types to Go friendly types
			switch t {
			case "int":
				t = "int32"
			case "float":
				t = "float32"
			}

			fmt.Fprintf(consts, "const %s %s = %d\n", l, t, v)
		}
	}

	return true
}

func main() {
	if !generate() {
		os.Exit(10)
	}

	fmt.Println("vertex and fragment shaders generated")
}
