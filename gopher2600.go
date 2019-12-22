package main

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/debugger"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/colorterm"
	"gopher2600/debugger/terminal/plainterm"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/gui/sdldebug"
	"gopher2600/gui/sdlplay"
	"gopher2600/modalflag"
	"gopher2600/paths"
	"gopher2600/performance"
	"gopher2600/playmode"
	"gopher2600/recorder"
	"gopher2600/regression"
	"gopher2600/television"
	"gopher2600/wavwriter"
	"io"
	"os"
	"strings"
)

const defaultInitScript = "debuggerInit"

func main() {
	// we generate random numbers in some places. seed the generator with the
	// current time
	// rand.Seed(int64(time.Now().Second()))

	md := &modalflag.Modes{Output: os.Stdout}
	md.NewArgs(os.Args[1:])
	md.NewMode()
	md.AddSubModes("RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS")

	p, err := md.Parse()
	switch p {
	case modalflag.ParseHelp:
		os.Exit(0)
	case modalflag.ParseError:
		fmt.Printf("* %s\n", err)
		os.Exit(10)
	}

	switch md.Mode() {
	case "RUN":
		fallthrough

	case "PLAY":
		err = play(md)

	case "DEBUG":
		err = debug(md)

	case "DISASM":
		err = disasm(md)

	case "PERFORMANCE":
		err = perform(md)

	case "REGRESS":
		err = regress(md)
	}

	if err != nil {
		fmt.Printf("* %s\n", err)
		os.Exit(20)
	}
}

func play(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	tvType := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	scaling := md.AddFloat64("scale", 3.0, "television scaling")
	stable := md.AddBool("stable", true, "wait for stable frame before opening display")
	fpscap := md.AddBool("fpscap", true, "cap fps to specification")
	record := md.AddBool("record", false, "record user input to a file")
	wav := md.AddString("wav", "", "record audio to wav file")
	patchFile := md.AddString("patch", "", "patch file to apply (cartridge args only)")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*tvType)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}
		defer tv.End()

		// add wavwriter mixer if wav argument has been specified
		if *wav != "" {
			aw, err := wavwriter.New(*wav)
			if err != nil {
				return errors.New(errors.PlayError, err)
			}
			tv.AddAudioMixer(aw)
		}

		scr, err := sdlplay.NewSdlPlay(tv, float32(*scaling))
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

		err = playmode.Play(tv, scr, *stable, *fpscap, *record, cartload, *patchFile)
		if err != nil {
			return err
		}
		if *record {
			fmt.Println("! recording completed")
		}
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func debug(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	tvType := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	termType := md.AddString("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
	initScript := md.AddString("initscript", paths.ResourcePath(defaultInitScript), "script to run on debugger start")
	profile := md.AddBool("profile", false, "run debugger through cpu profiler")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	tv, err := television.NewTelevision(*tvType)
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}
	defer tv.End()

	scr, err := sdldebug.NewSdlDebug(tv, 2.0)
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}

	// start debugger with choice of interface and cartridge
	var cons terminal.Terminal

	switch strings.ToUpper(*termType) {
	default:
		fmt.Printf("! unknown terminal type (%s) defaulting to plain\n", *termType)
		fallthrough
	case "PLAIN":
		cons = &plainterm.PlainTerminal{}
	case "COLOR":
		cons = &colorterm.ColorTerminal{}
	}

	dbg, err := debugger.NewDebugger(tv, scr, cons)
	if err != nil {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		runner := func() error {
			cartload := cartridgeloader.Loader{
				Filename: md.GetArg(0),
				Format:   *cartFormat,
			}
			err := dbg.Start(*initScript, cartload)
			if err != nil {
				return err
			}
			return nil
		}

		if *profile {
			err := performance.ProfileCPU("debug.cpu.profile", runner)
			if err != nil {
				return err
			}
			err = performance.ProfileMem("debug.mem.profile")
			if err != nil {
				return err
			}
		} else {
			err := runner()
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func disasm(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}
		dsm, err := disassembly.FromCartridge(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				dsm.Dump(md.Output)
			}

			return errors.New(errors.DisassemblyError, err)
		}
		dsm.Dump(md.Output)
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func perform(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	display := md.AddBool("display", false, "display TV output")
	fpscap := md.AddBool("fpscap", true, "cap FPS to specification (only valid if -display=true)")
	scaling := md.AddFloat64("scale", 3.0, "display scaling (only valid if -display=true")
	tvType := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	duration := md.AddString("duration", "5s", "run duration (note: there is a 2s overhead)")
	profile := md.AddBool("profile", false, "produce cpu and memory profiling reports")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*tvType)
		if err != nil {
			return err
		}
		defer tv.End()

		if *display {
			scr, err := sdlplay.NewSdlPlay(tv, float32(*scaling))
			if err != nil {
				return err
			}

			err = scr.(gui.GUI).SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return err
			}

			err = scr.(gui.GUI).SetFeature(gui.ReqSetFPSCap, *fpscap)
			if err != nil {
				return err
			}
		}

		err = performance.Check(md.Output, *profile, tv, *duration, cartload)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}

func regress(md *modalflag.Modes) error {
	md.NewMode()
	md.AddSubModes("RUN", "LIST", "DELETE", "ADD")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch md.Mode() {
	case "RUN":
		md.NewMode()

		// no additional arguments
		verbose := md.AddBool("verbose", false, "output more detail (eg. error messages)")
		failOnError := md.AddBool("fail", false, "fail on error")

		p, err := md.Parse()
		if p != modalflag.ParseContinue {
			return err
		}

		err = regression.RegressRunTests(md.Output, *verbose, *failOnError, md.RemainingArgs())
		if err != nil {
			return err
		}

	case "LIST":
		md.NewMode()

		// no additional arguments

		p, err := md.Parse()
		if p != modalflag.ParseContinue {
			return err
		}

		switch len(md.RemainingArgs()) {
		case 0:
			err := regression.RegressList(md.Output)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("no additional arguments required for %s mode", md)
		}

	case "DELETE":
		md.NewMode()

		answerYes := md.AddBool("yes", false, "answer yes to confirmation")

		p, err := md.Parse()
		if p != modalflag.ParseContinue {
			return err
		}

		switch len(md.RemainingArgs()) {
		case 0:
			return fmt.Errorf("database key required for %s mode", md)
		case 1:

			// use stdin for confirmation unless "yes" flag has been sent
			var confirmation io.Reader
			if *answerYes {
				confirmation = &yesReader{}
			} else {
				confirmation = os.Stdin
			}

			err := regression.RegressDelete(md.Output, confirmation, md.GetArg(0))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("only one entry can be deleted at at time when using %s mode", md)
		}

	case "ADD":
		return regressAdd(md)
	}

	return nil
}

func regressAdd(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	tvType := md.AddString("tv", "AUTO", "television specification: NTSC, PAL [cartridge args only]")
	numframes := md.AddInt("frames", 10, "number of frames to run [cartridge args only]")
	state := md.AddBool("state", false, "record TV state at every CPU step [cartrdige args only]")
	mode := md.AddString("mode", "video", "type of digest to create [cartridge args only]")
	notes := md.AddString("notes", "", "annotation for the database")

	md.AdditionalHelp("The regression test to be added can be the path to a cartrige file or a previously recorded playback file. For playback files, the flags marked [cartridge args only] do not make sense and will be ignored.")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge or playback file required for %s mode", md)
	case 1:
		var rec regression.Regressor

		if recorder.IsPlaybackFile(md.GetArg(0)) {
			// check and warn if unneeded arguments have been specified
			md.Visit(func(flg string) {
				if flg == "frames" {
					fmt.Printf("! ignored %s flag when adding playback entry\n", flg)
				}
			})

			rec = &regression.PlaybackRegression{
				Script: md.GetArg(0),
				Notes:  *notes,
			}
		} else {
			cartload := cartridgeloader.Loader{
				Filename: md.GetArg(0),
				Format:   *cartFormat,
			}

			// parse digest mode, failing if string is not recognised
			m, err := regression.ParseDigestMode(*mode)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			rec = &regression.DigestRegression{
				Mode:      m,
				CartLoad:  cartload,
				TVtype:    strings.ToUpper(*tvType),
				NumFrames: *numframes,
				State:     *state,
				Notes:     *notes,
			}
		}

		err := regression.RegressAdd(md.Output, rec)
		if err != nil {
			// using carriage return (without newline) at beginning of error
			// message because we want to overwrite the last output from
			// RegressAdd()
			return fmt.Errorf("\rerror adding regression test: %v", err)
		}
	default:
		return fmt.Errorf("regression tests can only be added one at a time")
	}

	return nil
}
