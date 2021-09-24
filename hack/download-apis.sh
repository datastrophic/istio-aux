#!/bin/bash

set -euox pipefail

echo "TEST"

check_requirement() {
  if ! [ -x "$(command -v ${1})" ]; then
  echo "Error: ${1} is not installed" >&2
  exit 1
  fi
}

check_requirement "git"
check_requirement "realpath"

project_root=$(dirname $(dirname $(realpath $0)))
target_dir="${project_root}/pkg/api/vendor"

operator_tag="${OPERATOR_TAG:-v1.3.0-rc.2}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf -- "${tmp_dir}"' EXIT

git clone --depth 1 --branch "${operator_tag}" https://github.com/kubeflow/tf-operator.git "${tmp_dir}"

rm -rf "${target_dir}"
mkdir -p "${target_dir}"

cp -r "${tmp_dir}/pkg/apis/." "${target_dir}"
