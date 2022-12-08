package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
    "strings"
)

type Config struct {
	Host      string `json:"host"`      // hostname for postgres db
	Port      int    `json:"port"`      // port database connector listens on
	Database  string `json:"database"`  // name of the database ('shrindex')
	User      string `json:"user"`      // username for database access
	Password  string `json:"password"`  // password associated with username
	query     string `json:"query"`     // query to use to fetch documents (default: evertying that isn't deleted)
	ChunkSize int    `json:"chunkSize"` // number of documents in each package
	SolrUrl   string `json:"solrUrl"`   // base url to solr collection
    RedisUrl  string `json:"redisUrl"`  // URL to Redis instance
    Authorities bool `json:"authorities"` // whether to do authority processing
	Workers   int    `json:"workers"`   // number of concurrent workers (cpu count -1)
    StartId   string `json:"startId"`  // document ID to start with when processing partial results
}

func (c *Config) QueryString() string {
    if len(c.StartId) == 0 {
        return c.query
    }
    return strings.Replace(c.query, "not deleted", "not deleted AND id >= :startId", 1)
}

func (c *Config) HasParameters() bool {
    return len(c.StartId) > 0
}
    

func (c *Config) DatabaseUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.Database)
}

func (c *Config) DisplayDatabaseUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", "[USER]", "[REDACTED]", c.Host, c.Port, c.Database)
}

// sets defaults and validates a configuration
// error will be non-nil if there is an unrecoverable problem
func (c *Config) Validate() error {
	if c.ChunkSize < 10 || c.ChunkSize > 100000 {
		return errors.New("chunkSize should be between 10 and 100000")
	}

	if c.Workers < 1 || c.Workers > runtime.NumCPU() {
		return errors.New("workers must be between 1 and " + strconv.Itoa(runtime.NumCPU()))
	}

	if c.Password == "" {
		return errors.New("configuration does not contain password (database)")
	}

	if c.SolrUrl == "" {
		return errors.New("configuration does not contain solrUrl")
	}
	return nil
}

// loads a configuration from a file; if no files are passed in,
// reads from config.json
func LoadConfig(files ...string) (*Config, error) {
	conf := &Config{
		Host:      "localhost",
		Port:      5432,
		User:      "shrindex",
		Database:  "shrindex",
		query:     "select id, txn_id, owner, content from documents WHERE NOT deleted ORDER BY id ASC",
		SolrUrl:   "http://localhost:8983/solr/trlnbib",
		ChunkSize: 20000,
		Workers:   runtime.NumCPU() - 1,
        Authorities: true,
        RedisUrl: "redis://localhost:6379/0",
        StartId: ""}
        
	if conf.Workers < 1 {
		conf.Workers = 1
	}
	filename := "config.json"
	if len(files) > 1 {
		filename = files[1]
	}

	log.Println("Loading configuration from", filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	err = conf.Validate()
	if err != nil {
		return nil, err
	}
	return conf, nil
}
