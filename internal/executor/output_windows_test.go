//go:build windows

package executor

import "testing"

func TestNormalizeWindowsOutputEncodings(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		cp   uint32
		want string
	}{
		{"UTF16LE", []byte{0xff, 0xfe, 'S', 0, 0xec, 0}, 0, "Sì"},
		{"UTF16BE", []byte{0xfe, 0xff, 0, 'S', 0, 0xec}, 0, "Sì"},
		{"UTF8", append([]byte{0xef, 0xbb, 0xbf}, []byte("Sì")...), 0, "Sì"},
		{"OEM850", []byte{'S', 0x8d}, 850, "Sì"},
		{"CP1252", []byte{'S', 0xec}, 1252, "Sì"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeWindowsOutput(tt.data, tt.cp); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestCleanPowerShellCLIXML(t *testing.T) {
	in := "#< CLIXML\r\n<Objs xmlns=\"http://schemas.microsoft.com/powershell/2004/04\"><S S=\"output\">Operazione_x000D__x000A_riuscita</S><S S=\"progress\">rumore</S><S S=\"Error\">Errore utile</S><S S=\"debug\">debug</S></Objs>"
	want := "Operazione\r\nriuscita\nErrore utile\n"
	if got := cleanPowerShellCLIXML(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCleanPowerShellCLIXMLMalformedIsPreserved(t *testing.T) {
	in := "#< CLIXML\r\n<Objs><S>diagnostic"
	if got := cleanPowerShellCLIXML(in); got != in {
		t.Fatalf("diagnostic lost: %q", got)
	}
}
