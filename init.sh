#!/bin/sh

set -e

# enable golang and ruby latest versions on Amazon Linux
if [ `command -v amazon-linux-extras` ]; then
    # latest available version at time of writing
    sudo amazon-linux-extras enable golang1.11
    sudo amazon-linux-extras enable ruby2.6
fi

# ensure all prereqs are installed

sudo yum -y groupinstall "Development Tools"

sudo yum -y install yajl git golang ruby ruby-devel rubygem-bundler jq

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

ln -s $(pwd) ~/go/src/reindexer
go test && go build driver.go
cp driver ~/bin
