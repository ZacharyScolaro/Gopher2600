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

package terminal

// Style is used to identify the category of text being sent to the
// Terminal.TermPrint() function. The terminal implementation can interpret
// this how it sees fit - the most likely treatment is to print different
// styles in different colours.
type Style int

// List of print styles
const (
	// input after it has been normalised by the parser
	StyleInput Style = iota

	// help information
	StyleHelp

	// terminal prompt
	StylePromptCPUStep
	StylePromptVideoStep
	StylePromptConfirm

	// non-error information from a command
	StyleFeedback

	// disassembly output at cpu cycle boundaries
	StyleCPUStep

	// disassembly output at video cycle boundaries
	StyleVideoStep

	// information about the machine
	StyleInstrument

	// non-error information from a command. distinct from StyleFeedback
	// because some terminal may need to do some additional output processing
	// (eg. inserting an additional newline)
	StyleFeedbackNonInteractive

	// information as a result of an error. errors can be generated by the
	// emulation or the debugger
	StyleError
)

// IncludeInScriptOutput returns true if print styles is to be included in the
// output of a script recording
func (sty Style) IncludeInScriptOutput() bool {
	switch sty {
	case StyleError, StylePromptCPUStep, StylePromptVideoStep, StylePromptConfirm:
		return false
	default:
		return true
	}
}

// IsPrompt returns true if the style is considered to be one of the prompt
// styles, false otherwise.
func (sty Style) IsPrompt() bool {
	switch sty {
	case StylePromptCPUStep, StylePromptVideoStep, StylePromptConfirm:
		return true
	default:
		return false
	}
}
