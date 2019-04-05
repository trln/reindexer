package ingest

import (
	"log"
	"os"
	"os/exec"
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
	combined, err := cmd.CombinedOutput()
	if combined != nil {
		log.Println("[", filename, "] argot output ", string(combined))
	}
	return err
}
