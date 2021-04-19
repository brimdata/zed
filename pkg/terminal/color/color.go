package color

import (
	"fmt"
)

var Enabled = false

type Code int

var (
	Reset      Code = -1
	Red        Code = 1
	Green      Code = 2
	GrayYellow Code = 3
	Blue       Code = 4
	Turqoise   Code = 31
	Purple     Code = 105
	Orange     Code = 208
	Pink       Code = 200
)

func (code Code) String() string {
	if Enabled {
		if code == Reset {
			return "\u001b[0m"
		}
		return fmt.Sprintf("\u001b[38;5;%dm", code)
	}
	return ""
}

func (code Code) Colorize(s string) string {
	if !Enabled {
		return s
	}
	return code.String() + s + Reset.String()
}

func Gray(level int) Code {
	if level < 0 {
		level = 0
	} else if level > 23 {
		level = 23
	}
	return Code(255 - level)
}

func Palette() string {
	var out string
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			code := i*16 + j
			out += Code(code).String()
			out += fmt.Sprintf(" %d", code)
		}
	}
	out += Reset.String()
	return out
}
