# TRLN Discovery Indexer

Standalone tool for (re)indexing TRLN Discovery documents from the shared
database into a Solr collection.

Useful to prepare a new Solr instance or SolrCloud cluster with documents that
expect a different structure than the current production index.

## Details

It runs a query against the "Spofford" database to pull out the Argot of all
current documents, outputs those into files of a (configurable) number of
records, then runs `argot ingest` on the resulting files. (see the `argot-ruby`
gem for more information on this process). This has the effect of reindexing
all the documents that match the original query.

Because this might involve a lot of documents and take some time, the driver is
written in Go and uses concurrent workers for the argot and Solr-related parts
of the process.

This process is designed primarily for indexing into a non-production
collection, to test new index configurations, or prepare for Solr upgrades.  It
may involve the use of different version of Argot or Solr than the ones used in
production.  If you want to reindex things in production, you are strongly
encouraged to use or adapt tools available in `trln-ingest` (spoffford) for
that.

## Building

If you are running this on either Amazon Linux or some flavour of Red Hat Enterprise Linux (including CentOS and Fedora), you should be able to install all the prequisites and build the `driver` program by simply running

    $ ./init.sh [optional argot branch]

In this directory.  If you're not running one of those flavours of Linux, use
the cues in that script as a starting point (mostly that will involve the name
of the OS' package manager and the names of the packages).

The build process will also pull down `argot-ruby` and build it; if you want to
use any version of argot other than the one in the `master` branch of that
repository, pass in the branch name as the argument to the init script.

As a final step, the script builds the go program and installs it to
`~/bin/reindexer` (as well as to `~/go/bin/reindexer`; the latter is less
likely to be on your path)

### Running

Once built, the `reindexer` program can be copied anywhere on the system and run
from there.  It takes one optional argument, which is the name of a
configuration file (format described below).  If omitted, it will load the
configuration from `config.json` in the current working directory.

### Configuration

The configuration file's format is JSON, and has the following structure (see `config/config.go` and the definition of the `Config` struct for more guidance):

```json
{
    "host" : "localhost",
    "port" : 5432,
    "database" : "shrindex",
    "user": "shrindex",
    "password" : "no default",
    "query" : "SQL query used to fetch documents",
    "chunkSize" : 20000,
    "solrUrl" : "http://localhost:8983/solr/trlnbib",
    "authorities": true,
    "redisUrl": "redis://localhost:6379/0",
    "workers" : 3,
    "startId": ""
}
```

Where the value in the sample above looks like a sensible value, it's the
default; you MUST provide at least `password`.  `workers` defaults to the
number simultaneous threads the current machine can run (never lower than 1).
The default value for `query` selects all non-deleted documents from the
database, and documents are retrieved in "asciibetical" sort order.

`startId` is provided in case the reindex process is interrupted, and you need
to start over; when set to a non-empty string, all documents where the `id`
field sorts *after* this value will be indexed.

You can query the Solr index for the 'last' ingested document ID via the
following parameters to the `select` handler:

    $ curl -s 'http://localhost:8983/solr/gettingstarted/select?q=*:*&fl=id&sort=id+desc&rows=1' | jq .response.docs[0].id`

(`jq` is installed by default by the ansible script used to install this
project).

Authority processing against a redis database (the `authorities` attribute)
is `true` by default. When `authorities` is `true`, the `redisUrl` will be
checked to see whether it's accessible, and if this check fails, the process
will abort.

The master process loops over the documents matching the query, and outputs
them into files with no more than `chunkSize` records in them.  At  that point,
it passes that file to an available worker, which then runs `argot ingest -s
[solrUlr] -a --redis-url [redisUrl]` on the file (assuming `authorities` is
true; if `false` omit the `-a`, `--redis-url` and redis URL parameters), which
flattens and suffixes the Argot records in the file, and then submits the
results to Solr for reindex.

The entire process is logged to STDERR, so you may want to run the driver program thusly:

    $ ~/go/bin/reindexer 2> ingest-log-$(date +%Y-%m-%d).log&

To run it in the background, and then poke into the log now and again to check
on progress.  Prepend this command with `nohup` to allow you to log off or get
disconnected without terminating the process.
