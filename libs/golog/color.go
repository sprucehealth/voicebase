package golog

type Color string

const (
	ColorNone    Color = ""
	ColorReset   Color = "\033[0m"
	ColorBold    Color = "\033[1m"
	ColorBlack   Color = "\033[0;30m"
	ColorRed     Color = "\033[0;31m"
	ColorGreen   Color = "\033[0;32m"
	ColorYellow  Color = "\033[0;33m"
	ColorBlue    Color = "\033[0;34m"
	ColorMagenta Color = "\033[0;35m"
	ColorCyan    Color = "\033[0;36m"

	ColorLightGray Color = "\033[0;37m"
	ColorGray      Color = "\033[1;30m"

	ColorBoldRed     Color = "\033[1;31m"
	ColorBoldGreen   Color = "\033[1;32m"
	ColorBoldYellow  Color = "\033[1;33m"
	ColorBoldBlue    Color = "\033[1;34m"
	ColorBoldMagenta Color = "\033[1;35m"
	ColorBoldCyan    Color = "\033[1;36m"
	ColorWhite       Color = "\033[1;37m"
)

type BgColor string

const (
	ColorBgNone    BgColor = ""
	ColorBgRed     BgColor = "\033[41m"
	ColorBgGreen   BgColor = "\033[42m"
	ColorBgYellow  BgColor = "\033[43m"
	ColorBgBlue    BgColor = "\033[44m"
	ColorBgMagenta BgColor = "\033[45m"
	ColorBgCyan    BgColor = "\033[46m"
	ColorBgWhite   BgColor = "\033[47m"
)
