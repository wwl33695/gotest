clear

set OLDGOPATH = $GOPATH

export GOPATH=`pwd`

go install sipclient

export GOPATH=$OLDGOPATH
unset OLDGOPATH
