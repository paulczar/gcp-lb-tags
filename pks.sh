#!/bin/bash

URL="http://metadata.google.internal/computeMetadata/v1/"
HEADER='Metadata-Flavor: Google'

PROJECT_ID=$(curl -sH "$HEADER" $URL/project/project-id)

FQ_NETWORK=$(curl -sH "$HEADER" $URL/instance/network-interfaces/0/network)
NETWORK=$(basename "$FQ_NETWORK")
UUID=$(curl -sH "$HEADER" "$URL/instance/tags?alt=text" | grep p-bosh-service-instance | sed 's/^p-bosh-service-instance-//')

FQ_ZONE=$(curl -sH "$HEADER" $URL/instance/zone)
REGION=$(basename "$FQ_ZONE" | sed 's/-.$//')

echo "network: $NETWORK"
echo "project: $PROJECT_ID"
echo "uuid: $UUID"
echo "region: $REGION"

/app/gcp-lb-tags create --loop --name pks-$UUID \
   --project $PROJECT_ID --network $NETWORK \
   --region $REGION --port 8443 --tags=service-instance-$UUID-master \
   --labels="deployment:service-instance-$UUID" --labels="job:master"