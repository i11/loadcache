#!/bin/bash -e

VERSION=""

usage() { echo "Usage: $0 [-t <major|minor|patch>] [-v <semver string (e.g. 0.1.0 or v0.1.0)>]" 1>&2; }

# get parameters
no_args="true"
while getopts ':t:v:' flag; do
  case "${flag}" in
    v) VERSION=${OPTARG};;
    t) TYPE=${OPTARG};;
    *) usage; exit 1;;
  esac
  no_args="false"
done

[[ "$no_args" == "true" ]] && { usage; exit 1; }
[[ "x${TYPE}" != "x" ]] && [[ $TYPE != 'major' ]] && [[ $TYPE != 'minor' ]] && [[ $TYPE != 'patch' ]] && { >&2 echo 'Incorrect type specified. Try using one of these: major, minor or patch'; usage; exit 1; }

by_type() {
  local TYPE="${1}"
  local CURRENT_VERSION="${2}"

  if [[ $CURRENT_VERSION == '' ]]; then
    CURRENT_VERSION='v0.1.0'
  fi

  # replace . with space so can split into an array
  local CURRENT_VERSION_PARTS=(${CURRENT_VERSION//./ })

  # get number parts
  local VNUM1=${CURRENT_VERSION_PARTS[0]##*v}
  local VNUM2=${CURRENT_VERSION_PARTS[1]}
  local VNUM3=${CURRENT_VERSION_PARTS[2]}
  local VPREFIX=${CURRENT_VERSION_PARTS[0]%${VNUM1}}

  if [[ $TYPE == 'major' ]]; then
    VNUM1=$((VNUM1+1))
    VNUM2=0
    VNUM3=0
  elif [[ $TYPE == 'minor' ]]; then
    VNUM2=$((VNUM2+1))
    VNUM3=0
  elif [[ $TYPE == 'patch' ]]; then
    VNUM3=$((VNUM3+1))
  else
    >&2 echo "No version type (https://semver.org/) or incorrect type specified, try: -v [major, minor, patch]"
    exit 4
  fi

  # new version
  local NEW_VERSION="${VPREFIX}${VNUM1}.${VNUM2}.${VNUM3}"
  echo $NEW_VERSION
}

by_version() {
  local NEW_VERSION="${1}"
  local CURRENT_VERSION="${2}"

  local NEW_VERSION_PARTS=(${NEW_VERSION//./ })
  local NEW_VNUM1=${NEW_VERSION_PARTS[0]##*v}
  local NEW_VNUM2=${NEW_VERSION_PARTS[1]}
  local NEW_VNUM3=${NEW_VERSION_PARTS[2]}

  local CURRENT_VERSION_PARTS=(${CURRENT_VERSION//./ })
  local VNUM1=${CURRENT_VERSION_PARTS[0]##*v}
  local VNUM2=${CURRENT_VERSION_PARTS[1]}
  local VNUM3=${CURRENT_VERSION_PARTS[2]}

  if [[ "x${NEW_VNUM1}" == 'x' ]] || [[ "x${NEW_VNUM1}" == 'x' ]] || [[ "x${NEW_VNUM1}" == 'x' ]]; then
    >&2 echo 'Incorrect version specified. Try using the type argument instead.'
    exit 3
  fi

  if [[ $NEW_VNUM1 -eq $VNUM1 ]] && [[ $NEW_VNUM2 -eq $VNUM2 ]] && [[ $NEW_VNUM3 -eq $VNUM3 ]]; then
    >&2 echo 'New specified vesion is the same as the current one.'
    exit 30
  fi

  if [[ $NEW_VNUM1 -lt $VNUM1 ]]; then
    >&2 echo 'Invalid specified version. The new major version is smaller than the current one.'
    exit 31
  fi

  if [[ $NEW_VNUM1 -eq $VNUM1 ]] && [[ $NEW_VNUM2 -lt $VNUM2 ]]; then
    >&2 echo 'Invalid specified version. The new minor version is smaller than the current one.'
    exit 32
  fi

  if [[ $NEW_VNUM2 -eq $VNUM2 ]] && [[ $NEW_VNUM3 -lt $VNUM3 ]]; then
    >&2 echo 'Invalid specified version. The new patch version is smaller than the current one.'
    exit 33
  fi

  echo $NEW_VERSION
}

main() {
  # get highest tag number, and add 1.0.0 if doesn't exist
  CURRENT_VERSION=$(git describe --tags $(git rev-list --tags --max-count=1) 2>/dev/null)
  echo "Current Version: $CURRENT_VERSION"

  NEW_TAG=''
  if [[ "x${VERSION}" != 'x' ]]; then
    NEW_TAG=$(by_version "${VERSION}" "${CURRENT_VERSION}")
  elif [[ "x${TYPE}" != 'x' ]]; then
    NEW_TAG=$(by_type "${TYPE}" "${CURRENT_VERSION}")
  fi

  if [[ "x${NEW_TAG}" == 'x' ]]; then
    echo "Something went horribly wrong... Sound the alarm"
    exit 2
  fi

  echo "Updating $CURRENT_VERSION to $NEW_TAG"

  # get current hash and see if it already has a tag
  GIT_COMMIT=$(git rev-parse HEAD)
  NEEDS_TAG=$(git describe --contains $GIT_COMMIT 2>/dev/null)

  # only tag if no tag already
  if [ -z "$NEEDS_TAG" ]; then
    git tag $NEW_TAG
    echo "Pushing $NEW_TAG tag"
    git push --tags

    local NEW_TAG_PARTS=(${NEW_TAG//./ })
    local NEW_BRANCH=${NEW_TAG_PARTS[0]}
    echo "Pushing $NEW_BRANCH branch"
    git push origin HEAD:$NEW_BRANCH --force-with-lease
  else
    echo "Already a tag on this commit"
  fi
  exit 0
}

main
