package console

// PrintProfile specifies the printing mode
type PrintProfile int

// enumeration of print profile types
const (
	// the following profiles are generated as a result of the emulation

	// disassembly output at cpu cycle boundaries
	CPUStep PrintProfile = iota

	// disassembly output at video cycle boundaries
	VideoStep

	// information about the machine
	MachineInfo

	// information about the emulator, rather than the emulated machine
	EmulatorInfo

	// the input prompt
	Prompt

	// non-error information from a command
	Feedback

	// help information
	Help

	// user input (not used by all user interface types [eg. echoing terminals])
	Input

	// information as a result of an error. errors can be generated by the
	// emulation or the debugger
	Error
)

// IncludeInScriptOutput returns true if print profile is to be included in the
// output of a script recording
func (pp PrintProfile) IncludeInScriptOutput() bool {
	switch pp {
	case Error, Input, Prompt:
		return false
	default:
		return true
	}
}
