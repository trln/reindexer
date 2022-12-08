#!/bin/sh

set -e

# we need the @Development tools group installed, along
# with ruby 2.6, golang, yajl, jq installed via yum

ARGOT_BRANCH=${1:-main}

[[ ! -d src ]] && mkdir src
pushd src
if [ ! -d argot-ruby ]; then
    git clone https://github.com/trln/argot-ruby.git
fi
cd argot-ruby
git checkout "${ARGOT_BRANCH}"

bundle install
# ensure that gem-installed tools are on the PATH
export PATH="${PATH}":~/bin:~/go/bin
bundle exec rake install
popd

go test && go install # creates '~/go/bin/reindexer'
cp ~/go/bin/reindexer ~/bin
