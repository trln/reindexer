#!/bin/sh

set -e

# enable golang and ruby latest versions on Amazon Linux
if [ `command -v amazon-linux-extras` ]; then
    # latest available version at time of writing
    sudo amazon-linux-extras enable golang1.11
    sudo amazon-linux-extras enable ruby2.4
fi

# ensure all prereqs are installed

sudo yum -y groupinstall "Development Tools"

sudo yum -y install yajl git golang ruby rubygem-bundler jq

ARGOT_BRANCH=${1:-master}

[[ ! -d src ]] && mkdir src
pushd src
if [ ! -d argot-ruby ]; then
    git clone https://github.com/trln/argot-ruby.git
fi
cd argot-ruby
git checkout "${ARGOT_BRANCH}"
bundle install
rake install
popd

go build driver.go
