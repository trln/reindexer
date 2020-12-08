#!/bin/sh

set -e

# we need the @Development tools group installed, along
# with ruby 2.6, golang, yajl, jq installed via yum

ARGOT_BRANCH=${1:-master}

[[ ! -d src ]] && mkdir src
pushd src
if [ ! -d argot-ruby ]; then
    git clone https://github.com/trln/argot-ruby.git
fi
cd argot-ruby
git checkout "${ARGOT_BRANCH}"
# these appear not to be installed on ARM systems
# and neither does rdoc
gem install io-console -N
gem install json -N
gem install rake -N

bundle install
# ensure that gem-installed tools are on the PATH
export PATH="${PATH}":~/bin
rake install
popd

go get github.com/jmoiron/sqlx
go get github.com/lib/pq

# need to symlink to build the golang stuff all packagey
SRCDIR=~/go/src/reindexer

if [ ! -L "${SRCDIR}" ]; then
        ln -s $(pwd) ${SRCDIR}
fi

go test && go build driver.go
cp driver ~/bin
