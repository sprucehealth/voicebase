package golog

type color string

const (
	colorNone    color = ""
	colorReset   color = "\033[0m"
	colorBold    color = "\033[1m"
	colorBlack   color = "\033[0;30m"
	colorRed     color = "\033[0;31m"
	colorGreen   color = "\033[0;32m"
	colorYellow  color = "\033[0;33m"
	colorBlue    color = "\033[0;34m"
	colorMagenta color = "\033[0;35m"
	colorCyan    color = "\033[0;36m"

	colorLightGray color = "\033[0;37m"
	colorGray      color = "\033[1;30m"

	colorBoldRed     color = "\033[1;31m"
	colorBoldGreen   color = "\033[1;32m"
	colorBoldYellow  color = "\033[1;33m"
	colorBoldBlue    color = "\033[1;34m"
	colorBoldMagenta color = "\033[1;35m"
	colorBoldCyan    color = "\033[1;36m"
	colorWhite       color = "\033[1;37m"
)

type bgColor string

const (
	colorBgNone    bgColor = ""
	colorBgRed     bgColor = "\033[41m"
	colorBgGreen   bgColor = "\033[42m"
	colorBgYellow  bgColor = "\033[43m"
	colorBgBlue    bgColor = "\033[44m"
	colorBgMagenta bgColor = "\033[45m"
	colorBgCyan    bgColor = "\033[46m"
	colorBgWhite   bgColor = "\033[47m"
)
