#!/bin/sh
trap "exit" SIGINT

echo Configured to generate new fortune every $INTERVAL seconds

while :
do
  echo $(date) Writing log...
  sleep $INTERVAL 
done
