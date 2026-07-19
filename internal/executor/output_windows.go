//go:build windows

package executor

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
)

var getOEMCP = syscall.NewLazyDLL("kernel32.dll").NewProc("GetOEMCP")

func normalizePlatformOutput(data []byte) string { return normalizeWindowsOutput(data, 0) }

// normalizeWindowsOutput converts redirected Windows output to UTF-8 and
// removes PowerShell's CLIXML transport envelope. A zero code page asks the OS.
func normalizeWindowsOutput(data []byte, codePage uint32) string {
	if len(data) == 0 {
		return ""
	}
	var text string
	switch {
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe:
		text = decodeUTF16(data[2:], false)
	case len(data) >= 2 && data[0] == 0xfe && data[1] == 0xff:
		text = decodeUTF16(data[2:], true)
	case utf16ZeroRatio(data, 1) >= .60:
		text = decodeUTF16(data, false)
	case utf16ZeroRatio(data, 0) >= .60:
		text = decodeUTF16(data, true)
	case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
		text = string(data[3:])
	case utf8.Valid(data):
		text = string(data)
	default:
		if codePage == 0 {
			cp, _, _ := getOEMCP.Call()
			codePage = uint32(cp)
		}
		text = decodeWindowsCodePage(data, codePage)
	}
	return cleanPowerShellCLIXML(strings.TrimPrefix(text, "\ufeff"))
}

func utf16ZeroRatio(b []byte, parity int) float64 {
	if len(b) < 4 {
		return 0
	}
	total, zeros := 0, 0
	for i := parity; i < len(b); i += 2 {
		total++
		if b[i] == 0 {
			zeros++
		}
	}
	return float64(zeros) / float64(total)
}

func decodeUTF16(b []byte, bigEndian bool) string {
	words := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		word := uint16(b[i]) | uint16(b[i+1])<<8
		if bigEndian {
			word = uint16(b[i])<<8 | uint16(b[i+1])
		}
		words = append(words, word)
	}
	return string(utf16.Decode(words))
}

func decodeWindowsCodePage(b []byte, codePage uint32) string {
	table := cp850
	if codePage == 1252 {
		table = cp1252
	}
	var out strings.Builder
	for _, c := range b {
		if c < utf8.RuneSelf {
			out.WriteByte(c)
		} else {
			out.WriteRune(table[c-128])
		}
	}
	return out.String()
}

// cleanPowerShellCLIXML retains normal and error records while dropping
// progress/debug/verbose noise. Malformed input is preserved for diagnostics.
func cleanPowerShellCLIXML(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "#< CLIXML") {
		return s
	}
	decoder := xml.NewDecoder(strings.NewReader(strings.TrimSpace(strings.TrimPrefix(trimmed, "#< CLIXML"))))
	var lines []string
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return s
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "S" {
			continue
		}
		stream := ""
		for _, attr := range start.Attr {
			if attr.Name.Local == "S" {
				stream = strings.ToLower(attr.Value)
			}
		}
		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return s
		}
		if stream == "progress" || stream == "debug" || stream == "verbose" {
			continue
		}
		if value = decodeCLIXMLEscapes(value); value != "" {
			lines = append(lines, value)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func decodeCLIXMLEscapes(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if i+7 <= len(s) && s[i:i+2] == "_x" && s[i+6] == '_' {
			if n, err := strconv.ParseUint(s[i+2:i+6], 16, 16); err == nil {
				out.WriteRune(rune(n))
				i += 7
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

var cp850 = [128]rune{
	0x00c7, 0x00fc, 0x00e9, 0x00e2, 0x00e4, 0x00e0, 0x00e5, 0x00e7, 0x00ea, 0x00eb, 0x00e8, 0x00ef, 0x00ee, 0x00ec, 0x00c4, 0x00c5,
	0x00c9, 0x00e6, 0x00c6, 0x00f4, 0x00f6, 0x00f2, 0x00fb, 0x00f9, 0x00ff, 0x00d6, 0x00dc, 0x00f8, 0x00a3, 0x00d8, 0x00d7, 0x0192,
	0x00e1, 0x00ed, 0x00f3, 0x00fa, 0x00f1, 0x00d1, 0x00aa, 0x00ba, 0x00bf, 0x00ae, 0x00ac, 0x00bd, 0x00bc, 0x00a1, 0x00ab, 0x00bb,
	0x2591, 0x2592, 0x2593, 0x2502, 0x2524, 0x00c1, 0x00c2, 0x00c0, 0x00a9, 0x2563, 0x2551, 0x2557, 0x255d, 0x00a2, 0x00a5, 0x2510,
	0x2514, 0x2534, 0x252c, 0x251c, 0x2500, 0x253c, 0x00e3, 0x00c3, 0x255a, 0x2554, 0x2569, 0x2566, 0x2560, 0x2550, 0x256c, 0x00a4,
	0x00f0, 0x00d0, 0x00ca, 0x00cb, 0x00c8, 0x0131, 0x00cd, 0x00ce, 0x00cf, 0x2518, 0x250c, 0x2588, 0x2584, 0x00a6, 0x00cc, 0x2580,
	0x00d3, 0x00df, 0x00d4, 0x00d2, 0x00f5, 0x00d5, 0x00b5, 0x00fe, 0x00de, 0x00da, 0x00db, 0x00d9, 0x00fd, 0x00dd, 0x00af, 0x00b4,
	0x00ad, 0x00b1, 0x2017, 0x00be, 0x00b6, 0x00a7, 0x00f7, 0x00b8, 0x00b0, 0x00a8, 0x00b7, 0x00b9, 0x00b3, 0x00b2, 0x25a0, 0x00a0,
}

var cp1252 = func() [128]rune {
	var t [128]rune
	for i := range t {
		t[i] = rune(i + 128)
	}
	t[0] = 0x20ac
	t[2] = 0x201a
	t[3] = 0x0192
	t[9] = 0x2030
	t[17] = 0x2018
	t[18] = 0x2019
	t[19] = 0x201c
	t[20] = 0x201d
	t[21] = 0x2022
	t[22] = 0x2013
	t[23] = 0x2014
	t[25] = 0x2122
	return t
}()
