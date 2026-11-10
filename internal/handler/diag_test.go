package handler

import (
	"encoding/json"
	"strings"
	"testing"
)

func diagMap(t *testing.T, s string) map[string]any {
	t.Helper()
	d, ok := parseDiag(json.RawMessage(s))
	if !ok {
		t.Fatalf("parseDiag(%s) falló", s)
	}
	return d
}

func TestParseDiag(t *testing.T) {
	cases := map[string]bool{
		``:                         false,
		`null`:                     false,
		`[1,2]`:                    false,
		`"x"`:                      false,
		`{"fw":"a1b2c3d","up":10}`: true,
		`{` + strings.Repeat(" ", maxDiagBytes) + `}`: false,
	}
	for in, want := range cases {
		if _, ok := parseDiag(json.RawMessage(in)); ok != want {
			t.Errorf("parseDiag(%.40q) = %v, want %v", in, ok, want)
		}
	}
}

func TestDiffDiag(t *testing.T) {
	base := `{"fw":"a1","up":100,"rr":1,"st":5,"ps":4,"ts":50,"tr":6,"hb":0,"gv":1,"fa":3,"gr":0,"bv":1}`

	if got := diffDiag(nil, diagMap(t, base)); len(got) != 1 || got[0] != "primer diagnóstico" {
		t.Errorf("sin previo = %v", got)
	}

	// Solo cambian campos no clave (up sube, ts, fa): nada que registrar.
	same := `{"fw":"a1","up":160,"rr":1,"st":5,"ps":4,"ts":110,"tr":6,"hb":0,"gv":1,"fa":5,"gr":0,"bv":1}`
	if got := diffDiag(diagMap(t, base), diagMap(t, same)); len(got) != 0 {
		t.Errorf("sin cambios clave = %v", got)
	}

	// Reinicio por energía, sin fix y con firmware nuevo.
	reboot := `{"fw":"b2","up":12,"rr":1,"st":5,"ps":3,"ts":2,"tr":5,"hb":0,"gv":0,"fa":-1,"gr":0,"bv":1}`
	got := strings.Join(diffDiag(diagMap(t, base), diagMap(t, reboot)), ", ")
	for _, want := range []string{"REINICIO (up 100→12, rr=1)", "fw a1→b2", "tr 6→5", "gv 1→0", "ps 4→3"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en %q", want, got)
		}
	}
}
