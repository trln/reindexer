package main

// first two packages must be installed via 'go get (package name)

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/trln/reindexer/config"
	"github.com/trln/reindexer/ingest"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
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

func NewIngestConfig(appConfig *config.Config) ingest.IngestConfig {
	return ingest.IngestConfig{
		SolrUrl:     appConfig.SolrUrl,
		RedisUrl:    appConfig.RedisUrl,
		Authorities: appConfig.Authorities}
}

func ingestWorker(id int, wg *sync.WaitGroup, configuration ingest.IngestConfig, files chan *os.File, errors chan<- error) {
	defer log.Printf("worker [%d] exited\n", id)
	for file := range files {
		log.Printf("[%d] received filename %s\n", id, file.Name())
		ingestErr := ingest.Ingest(file.Name(), configuration)
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

func getRows(db *sqlx.DB, conf *config.Config) (*sqlx.Rows, error) {
	if !conf.HasParameters() {
		rows, err := db.Queryx(conf.QueryString())
		return rows, err
	}
	rows, err := db.NamedQuery(conf.QueryString(), conf.StartId)
	return rows, err
}

func AcquireLockfile() bool {
	filename := "reindex.lock"
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		file.WriteString(fmt.Sprint(os.Getpid()))
		defer file.Close()
		return true
	}
	return false
}

func ReleaseLockFile() error {
	return os.Remove("reindex.lock")
}

func ArgotBinary() bool {
	path, err := exec.LookPath("argot")
	log.Println("'argot' path resolves to", path)
	return err == nil
}

func main() {
	if !ArgotBinary() {
		log.Fatal("'argot' binary not found on path")
	}

	conf, err := config.LoadConfig(os.Args...)
	if err != nil {
		log.Fatal("Missing or invalid configuration: ", err)
	}
	if !AcquireLockfile() {
		log.Fatal("reindex.lock exists, which may indicate another instance is already running. If a process exists with the PID from this file, stop it first before trying to start a new one.  Otherwise you may manually remove the file and start a new reindexer")
	}
	defer ReleaseLockFile()

	if conf.Authorities {
		log.Println("Authorities processing is enabled, checking redisUrl")
		opt, err := redis.ParseURL(conf.RedisUrl)
		if err != nil {
			log.Fatalf("Unable to connect to redis at %s: %r", conf.RedisUrl, err)
		}
		rdb := redis.NewClient(opt)
		ctx := context.Background()
		redisPing := rdb.Ping(ctx)
		if redisPing.Err() != nil {
			log.Fatalf("Unable to connect to redis at %s: %v", conf.RedisUrl, redisPing.Err())
		}
	}
	if conf.HasParameters() {
		log.Printf("Start ID is %s\n", conf.StartId)
	}

	log.Println("Reading documents from database at ", conf.Host)
	log.Println("Indexing into", conf.SolrUrl)
	connStr := conf.DatabaseUrl()
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal("Unable to connect to database '", conf.DisplayDatabaseUrl(), "', error: ", err)
	}
	log.Println("Connected to database")
	defer db.Close()
	ingestFiles := make(chan *os.File, 2)
	errors := make(chan error, 300)

	ingestConfig := NewIngestConfig(conf)
	log.Fatal(ingestConfig)

	// setup worker pool
	var wg sync.WaitGroup
	log.Printf("Starting %d argot => solr workers", conf.Workers)
	for w := 1; w <= conf.Workers; w++ {
		go ingestWorker(w, &wg, ingestConfig, ingestFiles, errors)
	}
	go errorWorker(errors)
	log.Print("Executing query; this can take a while when sorting results")
	queryString := conf.QueryString()
	if err != nil {
		log.Fatalf("Unable to prepare query `%s`: %r", queryString, err)
	}

	rows, err := getRows(db, conf)
	defer rows.Close()
	count := 0
	if err != nil {
		log.Fatalf("Unable to execute document query (%s): %r", conf.QueryString(), err)
	}
	doc := Document{}
	output, err := newOutputFile()
	defer output.Close()
	if err != nil {
		log.Fatal("Unable to open new output file: ", err)
	}

	log.Print("Now processing database results")
	for rows.Next() {
		err := rows.StructScan(&doc)
		if err != nil {
			log.Fatalf("Error reading document from database: %v", err)
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
