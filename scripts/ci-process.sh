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
#           $WORKSPACE/scripts/ci-process.sh"
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
        sudo apt-get update
        sudo apt-get -y install golang
    fi
    # Set up Go environment to treat this workspace as within GOPATH. 
    export GOPATH=`pwd`
    export GOBIN=$GOPATH/bin
    export PATH=~/.local/bin:$GOBIN:$PATH
    echo "PATH is: '$PATH'"
    mkdir -p $GOPATH/src/github.com/provideapp
    ln -f -s `pwd` $GOPATH/src/github.com/provideapp/goldmine
    echo "GOPATH is: $GOPATH"
    echo '....Go-Getting....'
    go get -v github.com/provideapp/goldmine # TODO: revisit -u, deps, vendorizing. 
    if hash golint 2>/dev/null
    then
        echo 'Using golint...' # No version command or flag
    else 
        echo 'Installing golint'
        go get -u golang.org/x/lint/golint
    fi
    go env
}

setup_deployment_tools() 
{
    if hash python 2>/dev/null
    then
        echo 'Using: ' 
        python --version
    else
        echo 'Installing python'
        sudo apt-get update
        sudo apt-get -y install python2.7
    fi
    if hash pip 2>/dev/null
    then
        echo 'Using' `pip --version`
    else
        echo 'Installing python'
        sudo apt-get update
        sudo apt-get -y install python-pip
    fi
    if hash aws 2>/dev/null
    then
        echo 'Using AWS CLI: ' 
        aws --version
    else
        echo 'Installing AWS CLI'
        pip install awscli --upgrade --user
    fi
    if hash docker 2>/dev/null
    then
        echo 'Using docker' `docker -v`
    else
        echo 'Installing docker'
        sudo apt-get update
        sudo apt-get install -y apt-transport-https \
                                ca-certificates \
                                software-properties-common
        sudo apt-get install -y docker
    fi
    if hash jq 2>/dev/null
    then
        echo 'Using' `jq --version`
    else
        echo 'Installing jq'
        sudo apt-get update
        sudo apt-get -y install jq
    fi
}

bootstrap_environment() 
{
    echo '....Setting up environment....'
    setup_go
    setup_deployment_tools
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

perform_deployment()
{
    if [[ -z "${ECR_REPOSITORY_NAME}" || -z "${ECS_CLUSTER}" || -z "${ECS_TASK_DEFINITION_FAMILY}" || -z "${ECS_SERVICE_NAME}" ]]
    then
        echo '....[PRVD] Skipping container deployment....'
    else
        DEFINITION_FILE=ecs-task-definition.json
        MUNGED_FILE=ecs-task-definition-UPDATED.json
        echo '....list-images....'
        ECR_IMAGE_DIGEST=$(aws ecr list-images --repository-name provide/goldmine | jq '.imageIds[0].imageDigest')
        echo '....describe-images....'
        ECR_IMAGE=$(aws ecr describe-images --repository-name "${ECR_REPOSITORY_NAME}" --image-ids imageDigest="${ECR_IMAGE_DIGEST}" | jq '.')
        echo '....describe-task-definition....'
        ECS_TASK_DEFINITION=$(aws ecs describe-task-definition --task-definition "${ECS_TASK_DEFINITION_FAMILY}" | jq '.taskDefinition | del(.taskDefinitionArn) | del(.revision) | del(.status) | del(.compatibilities) | del(.requiresAttributes)')
        echo '....file manipulation....'
        echo $ECS_TASK_DEFINITION > $DEFINITION_FILE
        sed -E "s/goldmine:[a-zA-Z0-9\.-]+/goldmine:${buildRef}/" "./${DEFINITION_FILE}" > "./${MUNGED_FILE}"
        echo '....register-task-definition....'
        ECS_TASK_DEFINITION_ID=$(aws ecs register-task-definition --family "${ECS_TASK_DEFINITION_FAMILY}" --cli-input-json "file://${MUNGED_FILE}" | jq '.taskDefinition.taskDefinitionArn' | sed -E 's/.*\/(.*)"$/\1/')
        echo '....update-service....'
        aws ecs update-service --cluster "${ECS_CLUSTER}" --service "${ECS_SERVICE_NAME}" --task-definition "${ECS_TASK_DEFINITION_ID}"
    fi
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
echo '....[PRVD] Analyzing...'
go vet
golint > reports/linters/golint.txt # TODO: add -set_exit_status once we clean current issues up. 
echo '....[PRVD] Testing....'
go test -v -race -cover -html=cover/coverage.cov -o coverage.html ./... # TODO: -msan (for Clang's MemorySanitizer)
echo '....[PRVD] Building....'
go build -v
echo '....[PRVD] Docker Build....'
sudo docker build -t provide/goldmine .
echo '....[PRVD] Docker Tag....'
sudo docker tag provide/goldmine:latest "085843810865.dkr.ecr.us-east-1.amazonaws.com/provide/goldmine:${buildRef}"
echo '....[PRVD] Docker Push....'
$(aws ecr get-login --no-include-email --region us-east-1)
sudo docker push "085843810865.dkr.ecr.us-east-1.amazonaws.com/provide/goldmine:${buildRef}"
echo '....[PRVD] AWS Deployment....'
perform_deployment

# Finalization
popd &>/dev/null
echo '....CI process completed....'