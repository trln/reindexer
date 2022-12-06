package ingest

import (
	"log"
	"os"
	"os/exec"
)

// configuration specifically for
// ingest operations.
type IngestConfig struct {
    SolrUrl string
    Authorities bool
    RedisUrl string
}

func Ingest(filename string, config IngestConfig ) error {
	log.Println("Ingesting ", filename)
	defer func() {
		err := os.Remove(filename)
		if err != nil {
			log.Fatal("could not delete", filename, err)
		}
	}()

	defer log.Println("Completed work on ", filename)
    var cmd *exec.Cmd
    if config.Authorities {
        cmd = exec.Command("argot", "ingest", "-a", "--redis-url", config.RedisUrl, "-s", config.SolrUrl, filename)
    } else {
	    cmd = exec.Command("argot", "ingest", "-s", config.SolrUrl, filename)
    }
	combined, err := cmd.CombinedOutput()
	if combined != nil {
		log.Println("[", filename, "] argot output ", string(combined))
	}
	return err
}
