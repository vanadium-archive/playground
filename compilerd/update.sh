#!/bin/bash

# Script to rebuild and deploy compilerd and the Docker image (builder) to the
# playground backends.
#
# Usage:
#   gcutil ssh --project google.com:veyron playground-master
#   sudo su - veyron
#   veyron update
#   bash $VEYRON_ROOT/veyron/go/src/veyron.io/playground/compilerd/update.sh

set -e
set -u

readonly DATE=$(date +"%Y%m%d-%H%M%S")
readonly DISK="pg-data-${DATE}"

function unmount() {
  sudo umount /mnt
  gcloud compute --project "google.com:veyron" instances detach-disk --disk=${DISK} $(hostname) --zone us-central1-a
}

trap cleanup INT TERM EXIT

function cleanup() {
  # Unset the trap so that it doesn't run again on exit.
  trap - INT TERM EXIT
  if [[ -e /mnt/compilerd ]]; then
    # The disk is still mounted on the master, which means it's not yet mounted
    # on any backends. It's safe to unmount and delete it.
    unmount
    gcloud compute --project "google.com:veyron" disks delete ${DISK} --zone "us-central1-a"
  fi
  sudo docker rm ${DISK} &> /dev/null || true
}

function main() {
  if [[ ! -e ~/.netrc ]]; then
    echo "Unable to access git, missing ~/.netrc"
    exit 1
  fi

  local ROLLING="1"
  if [[ $# -gt 0 && ("$1" == "--no-rolling") ]]; then
    local ROLLING="0"
  fi

  gcloud compute --project "google.com:veyron" disks create ${DISK} --size "200" --zone "us-central1-a" --source-snapshot "pg-data-20140702" --type "pd-standard"
  gcloud compute --project "google.com:veyron" instances attach-disk --disk=${DISK} $(hostname) --zone us-central1-a
  sudo mount /dev/sdb1 /mnt

  # Build the docker image.
  cd ${VEYRON_ROOT}/veyron/go/src/veyron.io/playground/builder
  cp ~/.netrc ./netrc
  sudo docker build --no-cache -t playground .
  rm -f ./netrc

  # Export the docker image to disk.
  sudo docker save -o /mnt/playground.tar.gz playground

  # TODO(sadovsky): Before deploying the new playground image, we should run it
  # with real input and make sure it works (produces the expected output).

  # Copy the compilerd binary from the docker image to the disk.
  # NOTE(sadovsky): The purpose of the following line is to create a container
  # out of the docker image, so that we can copy out the compilerd binary.
  # Annoyingly, the only way to create the container is to run the image.
  # TODO(sadovsky): Why don't we just build compilerd using "veyron go install"?
  sudo docker run --name=${DISK} playground &> /dev/null || true
  sudo docker cp ${DISK}:/usr/local/veyron/veyron/go/bin/compilerd /tmp
  sudo mv /tmp/compilerd /mnt/compilerd
  sudo docker rm ${DISK}

  # Detach the disk so the backends can mount it.
  unmount

  # Update the template to use the new disk.
  cd ../compilerd
  sed -i -e s/pg-data-20140820/${DISK}/ pool_template.json
  gcloud preview replica-pools --zone=us-central1-a update-template --template=pool_template.json playground-pool
  git checkout -- pool_template.json

  # Perform a rolling restart of all the replicas.
  INSTANCES=$(gcloud preview replica-pools --zone=us-central1-a replicas --pool=playground-pool list|grep name:|cut -d: -f2)
  for i in ${INSTANCES}; do
    gcloud preview replica-pools --zone=us-central1-a replicas --pool=playground-pool restart ${i}
    if [[ "$ROLLING" == "1" ]]; then
      sleep 5m
    fi
  done
}

main "$@"
