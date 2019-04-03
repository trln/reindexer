package ingest

import (
    "log"
    "os"
    "os/exec"
    "io/ioutil"
)

func Ingest(filename string, solrUrl string) error {
    log.Println("Ingesting ", filename)
    defer func() {
        err := os.Remove(filename)
        if err != nil {
            log.Fatal("could not delete", filename, err)
        }
    }()

    defer log.Println("Completed work on ", filename)
    cmd := exec.Command("argot", "ingest", "-s", solrUrl, filename)
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return err
    }
    if err := cmd.Start(); err != nil {
        return err
    }
    slurp, _ := ioutil.ReadAll(stderr)
    output := string(slurp)
    if len(output) > 0 {
        log.Println("Argot output: ", output)
    }
    if err = cmd.Wait(); err != nil {
        return err
    }
    return nil
}
