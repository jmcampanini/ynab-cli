package cmd

import (
	"strings"
	"testing"
)

func TestWriteTablePadsByRuneCount(t *testing.T) {
	var out strings.Builder
	columns := []column{{name: "NAME"}, {name: "BALANCE", right: true}, {name: "NOTE"}}
	rows := [][]cell{
		{plain("Épargne"), plain("-1.234,56 €"), plain("x")},
		{plain("Cash"), plain("5,00 €"), plain("y")},
	}

	if err := writeTable(&out, columns, rows); err != nil {
		t.Fatal(err)
	}

	want := "NAME         BALANCE  NOTE\nÉpargne  -1.234,56 €  x\nCash          5,00 €  y\n"
	if out.String() != want {
		t.Errorf("writeTable() =\n%s\nwant\n%s", out.String(), want)
	}
}
