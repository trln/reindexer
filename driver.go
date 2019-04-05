package main

// first two packages must be installed via 'go get (package name)

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reindexer/config"
	"reindexer/ingest"
	"sync"
)

type Document struct {
	Id      string `db:"id"`
	TxnId   string `db:"txn_id"`
	Owner   string `db:"owner"`
	Content string `db:"content"`
}

func (doc *Document) Store() error {
	storeDir := path.Join("document", doc.Owner, doc.TxnId)
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		createError := os.MkdirAll(storeDir, os.ModePerm)
		if createError != nil {
			return createError
		}
	}
	storeFile := path.Join(storeDir, doc.Id+".json")
	log.Println("File storage path:", storeFile)
	return nil
}

func newOutputFile() (*os.File, error) {
	handle, err := ioutil.TempFile("", "argot-tmp.*.json")
	if err != nil {
		return nil, err
	}
	return handle, nil
}

func finishFile(f *os.File, wg *sync.WaitGroup, files chan *os.File) error {
	err := f.Close()
	if err != nil {
		return err
	}
	wg.Add(1)
	files <- f
	return nil
}

func ingestWorker(id int, wg *sync.WaitGroup, solrUrl string, files chan *os.File, errors chan<- error) {
	defer log.Printf("worker [%d] exited\n", id)
	for file := range files {
		log.Printf("[%d] received filename %s\n", id, file.Name())
		ingestErr := ingest.Ingest(file.Name(), solrUrl)
		log.Printf("[%d] external processing complete for %s\n", id, file.Name())
		if ingestErr != nil {
			errors <- ingestErr
		}
		wg.Done()
	}
}

func errorWorker(errors chan error) {
	for error := range errors {
		log.Println("[ERROR]", error)
	}
}

func main() {
	conf, err := config.LoadConfig(os.Args...)
	if err != nil {
		log.Fatal("Missing or invalid configuration: ", err)
	}
	log.Println("Reading documents from", conf.Host)
	log.Println("Indexing into", conf.SolrUrl)
	connStr := conf.DatabaseUrl()
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal("Unable to connect to database '", conf.DisplayDatabaseUrl(), "', error: ", err)
	}
	defer db.Close()
	ingestFiles := make(chan *os.File, 2)
	errors := make(chan error, 300)

	// setup worker pool
	var wg sync.WaitGroup
	log.Printf("Starting %d argot => solr workers", conf.Workers)
	for w := 1; w <= conf.Workers; w++ {
		go ingestWorker(w, &wg, conf.SolrUrl, ingestFiles, errors)
	}
	go errorWorker(errors)
	rows, err := db.Queryx(conf.Query)
	defer rows.Close()
	count := 0
	if err != nil {
		log.Fatal("Unable to execute document query("+conf.Query+"): ", err)
	}
	doc := Document{}
	output, err := newOutputFile()
	defer output.Close()
	if err != nil {
		log.Fatal("Unable to open new output file: ", err)
	}

	for rows.Next() {
		err := rows.StructScan(&doc)
		if err != nil {
			log.Fatal("Error reading document from database: ", err)
		}
		output.WriteString(doc.Content)
		count++
		if count%conf.ChunkSize == 0 {
			if err = finishFile(output, &wg, ingestFiles); err != nil {
				log.Fatal("Unable to ingest ", output.Name(), ": ", err)
			}
			if output, err = newOutputFile(); err != nil {
				log.Fatal("Unable to open new output file: ", err)
			}
		}
	}
	if output != nil {
		if err := finishFile(output, &wg, ingestFiles); err != nil {
			log.Fatal(err)
		}
	}
	close(ingestFiles)
	wg.Wait()
	select {
	case err := <-errors:
		log.Println("Error encountered during processing:", err.Error())
	default:
	}
}
