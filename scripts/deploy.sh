#!/usr/bin/env bash

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

. $SCRIPTS_DIR/build.sh
cp $DIR/README.md $BUILD_DIR/README.md

. $SCRIPTS_DIR/.env

echo "Stop"
ssh -p $SSH_PORT $REMOTE_IP -l root "systemctl stop $SERVICE_NAME"
echo "Send"
rsync -Pav -e "ssh -p $SSH_PORT -i $SCRIPTS_DIR/keys/key" $BUILD_DIR/ $REMOTE_USER@$REMOTE_IP:~/
echo "Start"
ssh -p $SSH_PORT $REMOTE_IP -l root "systemctl start $SERVICE_NAME"
sleep 4
ssh -p $SSH_PORT $REMOTE_IP -l root "systemctl status $SERVICE_NAME"