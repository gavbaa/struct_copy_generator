package main

import (
	"bytes"
	"os/exec"
	"strings"
)

func gofmt(s string) (string, error) {
	cmd := exec.Command("gofmt")
	cmd.Stdin = strings.NewReader(s)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
