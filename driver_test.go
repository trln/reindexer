package main

import (
    "os"
    "testing"
)

func TestNewOutputFile(t *testing.T) {
    output , err := newOutputFile()
    if output != nil {
        defer func() {
            os.Remove(output.Name())
        }()
    }
    if err != nil {
        t.Errorf("Unable to create new output file: %s", err)
    }
}
