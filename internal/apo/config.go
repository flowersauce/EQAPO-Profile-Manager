// Package apo parses and edits only eqm's managed portion of config.txt.
package apo

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
)

const (
	Begin     = "# EQAPO-Profile-Manager BEGIN"
	End       = "# EQAPO-Profile-Manager END"
	Separator = "# ------------------------------------------------------------"
)

type Document struct {
	Data       []byte
	Managed    bool
	Filename   string
	Enabled    bool
	start, end int
}

type line struct {
	text       string
	start, end int
}

func lines(data []byte) []line {
	var result []line
	for start := 0; start < len(data); {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start + 1
		}
		result = append(result, line{strings.TrimSuffix(strings.TrimSuffix(string(data[start:end]), "\n"), "\r"), start, end})
		start = end
	}
	return result
}

func ValidateFilename(name string) error {
	if !utf8.ValidString(name) || len(name) == 0 || !strings.HasSuffix(strings.ToLower(name), ".txt") {
		return fault.New("filename")
	}
	stem := name[:len(name)-4]
	if stem == "" || strings.EqualFold(stem, "None") || strings.TrimSpace(stem) == "" || strings.HasSuffix(stem, ".") || strings.HasSuffix(stem, " ") {
		return fault.New("filename")
	}
	for _, r := range name {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return fault.New("filename")
		}
	}
	base := strings.ToUpper(strings.Split(stem, ".")[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" {
		return fault.New("filename")
	}
	if len([]rune(base)) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && strings.ContainsRune("123456789¹²³", []rune(base)[3]) {
		return fault.New("filename")
	}
	return nil
}

func Parse(data []byte) (Document, error) {
	d := Document{Data: bytes.Clone(data)}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return d, fault.New("encoding")
	}
	content := bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	offset := len(data) - len(content)
	ls := lines(content)
	b, e := -1, -1
	for i, l := range ls {
		switch strings.TrimSpace(l.text) {
		case Begin:
			if b >= 0 {
				return d, damaged()
			}
			b = i
		case End:
			if e >= 0 {
				return d, damaged()
			}
			e = i
		}
	}
	if b < 0 && e < 0 {
		return d, nil
	}
	if b < 0 || e <= b {
		return d, damaged()
	}
	for _, l := range ls[:b] {
		if l.text != Separator && l.text != "# ============================================================================" {
			return d, damaged()
		}
	}
	count := 0
	for _, l := range ls[b+1 : e] {
		t := strings.TrimSpace(l.text)
		enabled := !strings.HasPrefix(t, "#")
		if !enabled {
			t = strings.TrimSpace(strings.TrimPrefix(t, "#"))
		}
		if !strings.HasPrefix(t, "Include:") {
			if enabled && t != "" {
				return d, damaged()
			}
			continue
		}
		count++
		target := strings.TrimSpace(strings.TrimPrefix(t, "Include:"))
		if target == "" {
			if enabled {
				return d, damaged()
			}
		} else {
			const prefix = `eqm-profiles\`
			if !strings.HasPrefix(target, prefix) {
				return d, damaged()
			}
			d.Filename = strings.TrimPrefix(target, prefix)
			if ValidateFilename(d.Filename) != nil {
				return d, damaged()
			}
		}
		d.Enabled = enabled
		d.start = l.start + offset
		// Preserve the existing newline outside the replacement span.
		d.end = d.start + len(l.text)
	}
	if count != 1 {
		return d, damaged()
	}
	d.Managed = true
	return d, nil
}

func damaged() error { e := fault.New("managed"); e.Code = 3; return e }

func (d Document) Select(filename string, enabled bool) ([]byte, error) {
	if !d.Managed {
		return nil, damaged()
	}
	if filename != "" {
		if err := ValidateFilename(filename); err != nil {
			return nil, err
		}
	} else if enabled {
		return nil, damaged()
	}
	s := "Include:"
	if filename != "" {
		s += ` eqm-profiles\` + filename
	}
	if !enabled {
		s = "# " + s
	}
	result := bytes.Clone(d.Data[:d.start])
	result = append(result, s...)
	result = append(result, d.Data[d.end:]...)
	return result, nil
}

// Takeover retains the original text as comments and initially selects None.
func Takeover(data []byte) ([]byte, error) {
	d, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if d.Managed {
		return bytes.Clone(data), nil
	}
	nl := "\r\n"
	if bytes.Contains(data, []byte("\n")) && !bytes.Contains(data, []byte("\r\n")) {
		nl = "\n"
	}
	var out strings.Builder
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		out.WriteString("\xef\xbb\xbf")
		data = data[3:]
	}
	out.WriteString(strings.Join([]string{Separator, Begin, "# Include:", End, Separator, "", "# Original config.txt"}, nl) + nl)
	for _, l := range lines(data) {
		out.WriteString("# ")
		out.Write(data[l.start:l.end])
	}
	// Do not add a footer: this preserves the original last line's newline state.
	return []byte(out.String()), nil
}
