// This library is adapted from the Go time pkg

module.exports = {
	Nanosecond: 1,
	Microsecond: 1e3,
	Millisecond: 1e6,
	Second: 1e9,
	Minute: 1e9 * 60,
	Hour: 1e9 * 60 * 60,
	Day: 1e9 * 60 * 60 * 24,

	// parseDuration parses a duration string.
	// A duration string is a possibly signed sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	parseDuration: function(s: string): {d: number; err: ?error} {
		// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
		var orig = s
		var f = 0.0
		var neg = false

		// Consume [-+]?
		if (s != "") {
			var c = s[0]
			if ((c == '-') || (c == '+')) {
				neg = c == '-'
				s = s.substr(1)
			}
		}
		// Special case: if all that is left is "0", this is zero.
		if (s == "0") {
			return {d:0, err:null}
		}
		if (s == "") {
			return {d:0, err:"invalid duration " + orig}
		}
		while (s != "") {
			var g = 0.0 // this element of the sequence

			// The next character must be [0-9.]
			if (!(s[0] == '.' || ('0' <= s[0] && s[0] <= '9'))) {
				return {d:0, err:"invalid duration " + orig}
			}
			// Consume [0-9]*
			var pl = s.length
			var xs = leadingInt(s)
			if (xs.err) {
				return {d:0, err:"invalid duration " + orig}
			}
			g = xs.x
			s = xs.rem
			var pre = pl != s.length // whether we consumed anything before a period

			// Consume (\.[0-9]*)?
			var post = false
			if (s != "" && s[0] == '.') {
				s = s.substr(1)
				var pl = s.length
				xs = leadingInt(s)
				if (xs.err) {
					return {d:0, err:"invalid duration " + orig}
				}
				s = xs.rem
				var scale = 1.0
				for (var n = pl - s.length; n > 0; n--) {
					scale *= 10
				}
				g += Math.floor(xs.x / scale)
				post = pl != s.length
			}
			if (!pre && !post) {
				// no digits (e.g. ".s" or "-.s")
				return {d:0, err:"invalid duration " + orig}
			}

			// Consume unit.
			var i = 0
			for (; i < s.length; i++) {
				var c = s[i]
				if (c == '.' || ('0' <= c && c <= '9')) {
					break
				}
			}
			if (i == 0) {
				return {d:0, err:"missing unit in duration " + orig}
			}
			var u = s.substr(0, i)
			s = s.substr(i)
			var unit = unitMap[u]
			if (!unit) {
				return {d:0, err:"unknown unit " + u + " in duration " + orig}
			}

			f += g * unit
		}
		if (neg) {
			f = -f
		}
		return {d:f, err:null}
	},

	// formatDuration returns a string representing the duration in the form "72h3m0.5s".
	// Leading zero units are omitted.  As a special case, durations less than one
	// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
	// that the leading digit is non-zero.  The zero duration formats as 0,
	// with no unit.
	formatDuration: function(d: number): string {
		// Largest time is 2540400h10m10.000000000s
		var buf = [
			"1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
			"1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
			"1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
			"1", "2"]
		var w = buf.length

		var u = d
		var neg = d < 0
		if (neg) {
			u = -u
		}

		if (u < module.exports.Second) {
			// Special case: if duration is smaller than a second,
			// use smaller units, like 1.2ms
			var prec = 0
			w--
			buf[w] = 's'
			w--
			if (u == 0) {
				return "0"
			} else if (u < module.exports.Microsecond) {
				// print nanoseconds
				prec = 0
				buf[w] = 'n'
			} else if (u < module.exports.Millisecond) {
				// print microseconds
				prec = 3
				// U+00B5 'µ' micro sign == 0xC2 0xB5
				buf[w] = 'µ'
			} else {
				// print milliseconds
				prec = 6
				buf[w] = 'm'
			}
			var wu = fmtFrac(buf, w, u, prec)
			w = wu.nw
			u = wu.nv
			w = fmtInt(buf, w, u)
		} else {
			w--
			buf[w] = 's'

			var wu = fmtFrac(buf, w, u, 9)
			u = wu.nv
			if (w != wu.nw || u%60 != 0) {
				w = wu.nw

				// u is now integer seconds
				w = fmtInt(buf, w, u%60)
			} else {
				w++
			}
			u = Math.floor(u/60)

			// u is now integer minutes
			if (u > 0) {
				var m = u%60
				if (m != 0) {
					w--
					buf[w] = 'm'
					w = fmtInt(buf, w, m)
				}
				u = Math.floor(u/60)

				// u is now integer hours
				if (u > 0) {
					var h = u%24
					if (h != 0) {
						w--
						buf[w] = 'h'
						w = fmtInt(buf, w, h)
					}
					u = Math.floor(u/24)

					// u is now integer days
					// Days can be different lengths but we don't really care
					// for our use case
					if (u > 0) {
						w--
						buf[w] = 'd'
						w = fmtInt(buf, w, u)
					}
				}
			}
		}

		if (neg) {
			w--
			buf[w] = '-'
		}

		return buf.slice(w).join("")
	},

	isTimestampBeforeNow: function(epoch: number): any {
		return epoch < ((new Date).getTime()/1000)
	},
}

var unitMap = {
	"ns": module.exports.Nanosecond,
	"us": module.exports.Microsecond,
	"µs": module.exports.Microsecond, // U+00B5 = micro symbol
	"μs": module.exports.Microsecond, // U+03BC = Greek letter mu
	"ms": module.exports.Millisecond,
	"s":  module.exports.Second,
	"m":  module.exports.Minute,
	"h":  module.exports.Hour,
	"d":  module.exports.Day,
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros.  it omits the decimal
// point too when the fraction is 0.  It returns the index where the
// output bytes begin and the value v/10**prec.
function fmtFrac(buf: Array<string>, w: number, v: number, prec: number): {nw: number; nv: number} {
	// Omit trailing zeros up to and including decimal point.
	var print = false
	for (var i = 0; i < prec; i++) {
		var digit = v % 10
		print = print || digit != 0
		if (print) {
			w--
			buf[w] = String.fromCharCode(digit + '0'.charCodeAt(0))
		}
		v = Math.floor(v/10)
	}
	if (print) {
		w--
		buf[w] = '.'
	}
	return {nw:w, nv:v}
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
function fmtInt(buf: Array<string>, w: number, v: number): number {
	if (v == 0) {
		w--
		buf[w] = '0'
	} else {
		while (v > 0) {
			w--
			buf[w] = String.fromCharCode((v%10) + '0'.charCodeAt(0))
			v = Math.floor(v/10)
		}
	}
	return w
}

// leadingInt consumes the leading [0-9]* from s.
function leadingInt(s: string): {x: number; rem: string; err: ?error} {
	var i = 0
	var x = 0
	for (; i < s.length; i++) {
		var c = s[i]
		if (c < '0' || c > '9') {
			break
		}
		if (x >= (1<<63-10)/10) {
			// overflow
			return {x:0, rem:"", err:"bad [0-9]*"}
		}
		x = x*10 + c.charCodeAt(0) - '0'.charCodeAt(0)
	}
	return {x:x, rem:s.substr(i), err:null}
}