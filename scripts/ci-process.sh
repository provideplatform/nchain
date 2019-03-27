#!/bin/bash
# Script for Continuous Integration
# Example Jenkins usage: 
#       /bin/bash -c \
#           "AWS_ACCESS_KEY_ID=xyz \
#           AWS_SECRET_ACCESS_KEY=abc \
#           AWS_DEFAULT_REGION=us-east-1 \
#           AWS_DEFAULT_OUTPUT=json \
#           ECR_REPOSITORY_NAME=provide/goldmine \
#           ECS_TASK_DEFINITION_FAMILY=goldmine-fargate \
#           ECS_CLUSTER=production \
#           ECS_SERVICE_NAME=goldmine \
#           '$WORKSPACE/scripts/ci-process.sh'"
set -o errexit # set -e
set -o nounset # set -u
set -o pipefail
# set -o verbose
trap die ERR
die() 
{
    echo "Failed at line $BASH_LINENO"; exit 1
}
echo Executing $0 $*

setup_go() 
{
    if hash go 2>/dev/null
    then
        echo 'Using' `go version`
    else
        echo 'Installing go'
        wget https://dl.google.com/go/go1.11.linux-amd64.tar.gz
        sudo tar -xvf go1.11.linux-amd64.tar.gz
        sudo mv go /usr/lib/go-1.11
        sudo ln -s /usr/lib/go-1.11 /usr/lib/go
        sudo ln -s /usr/lib/go-1.11/bin/go /usr/bin/go
        sudo ln -s /usr/lib/go-1.11/bin/gofmt /usr/bin/gofmt
    fi

    # Set up Go environment to treat this workspace as within GOPATH. 
    export GOPATH=`pwd`
    export GOBIN=$GOPATH/bin
    export PATH=~/.local/bin:$GOBIN:$PATH
    echo "PATH is: '$PATH'"
    mkdir -p $GOPATH/src/github.com/provideapp
    ln -f -s `pwd` $GOPATH/src/github.com/provideapp/goldmine
    echo "GOPATH is: $GOPATH"
    mkdir -p $GOBIN

    if hash glide 2>/dev/null
    then
        echo 'Using glide...'
    else 
        echo 'Installing glide...'
        curl https://glide.sh/get | sh
    fi

    go env
}

bootstrap_environment() 
{
    echo '....Setting up environment....'
    setup_go
    mkdir -p reports/linters
    echo '....Environment setup complete....'
}

get_build_info()
{
    echo '....Getting build values....'
    revNumber=$(echo `git rev-list HEAD | wc -l`) # the echo trims leading whitespace
    gitHash=`git rev-parse --short HEAD`
    gitBranch=`git rev-parse --abbrev-ref HEAD`
    buildDate=$(date '+%m.%d.%y')
    buildTime=$(date '+%H.%M.%S')
    echo "$(echo `git status` | grep "nothing to commit" > /dev/null 2>&1; if [ "$?" -ne "0" ]; then echo 'Local git status is dirty'; fi )";
    buildRef=${gitBranch}-${gitHash}-${buildDate}-${buildTime}
    echo 'Build Ref =' $buildRef
}

# Preparation
echo '....Running the full continuous integration process....'
scriptDir=`dirname $0`
pushd ${scriptDir}/.. &>/dev/null
echo 'Working Directory =' `pwd`

# The Process
echo '....[PRVD] Setting Up....'
bootstrap_environment
get_build_info
rm ./goldmine 2>/dev/null || true # silence error if not present
go fix .
go fmt
go clean -i

glide install
(cd vendor/ && tar c .) | (cd src/ && tar xf -)
rm -rf vendor/

make lint > reports/linters/golint.txt # TODO: add -set_exit_status once we clean current issues up. 

DATABASE_USER=postgres DATABASE_PASSWORD=postgres make test

go build -v
make ecs_deploy

popd &>/dev/null
echo '....CI process completed....'
