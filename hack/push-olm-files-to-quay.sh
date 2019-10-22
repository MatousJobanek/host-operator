#!/bin/bash

user_help () {
    echo "Generate ClusterServiceVersion and additional deployment files for openshift-marketplace"
    echo "options:"
    echo "-pr, --project-root      path to the root of the project the CSV should be generated for/in"
    exit 0
}

if [[ $# -lt 2 ]]
then
    user_help
fi

while test $# -gt 0; do
       case "$1" in
            -h|--help)
                user_help
                ;;
            -pr|--project-root)
                shift
                PRJ_ROOT_DIR=$1
                shift
                ;;
            -cv|--current-version)
                shift
                CURRENT_CSV_VERSION=$1
                shift
                ;;
            -nv|--next-version)
                shift
                NEXT_CSV_VERSION=$1
                shift
                ;;
            *)
               echo "$1 is not a recognized flag!" >> /dev/stderr
               user_help
               exit -1
               ;;
      esac
done

set -e

if [[ -z PRJ_ROOT_DIR ]]; then
    echo "--project-root parameter is not specified" >> /dev/stderr
    user_help
    exit 1;
fi

# Files and directories related vars
PRJ_NAME=`basename ${PRJ_ROOT_DIR}`
PKG_DIR=${PRJ_ROOT_DIR}/deploy/olm-catalog/${PRJ_NAME}
QUAY_NAMESPACE=${QUAY_NAMESPACE:"codeready-toolchain"}
GIT_COMMIT_ID=`git rev-parse --short HEAD`
PREVIOUS_GIT_COMMIT_ID=`git rev-parse --short HEAD`
TMP_FLATTEN_DIR="/tmp/${PRJ_NAME}_${GIT_COMMIT_ID}_flatten"

echo "## Pushing the OperatorHub package '${PRJ_NAME}' to the Quay.io '${QUAY_NAMESPACE}' organization"

echo "   - Flatten package to temporary folder: ${TMP_FLATTEN_DIR}"

rm -Rf "${TMP_FLATTEN_DIR}" > /dev/null 2&>1
mkdir -p "${TMP_FLATTEN_DIR}"
operator-courier flatten "${PKG_DIR}" ${TMP_FLATTEN_DIR}

NEW_VERSION="0.0.$(git rev-list --all --count)-${GIT_COMMIT_ID}"
echo "   - Push flattened files to Quay.io namespace '${QUAY_NAMESPACE}' as version ${NEW_VERSION}"

if [ -z "${QUAY_USERNAME}" ] || [ -z "${QUAY_PASSWORD}" ]
then
echo "#### ERROR: "
echo "You should have set ${QUAY_USERNAME_PLATFORM_VAR} and ${QUAY_PASSWORD_PLATFORM_VAR} environment variables"
echo "with a user that has write access to the following Quay.io namespace: ${quayNamespace}"
echo "or QUAY_USERNAME and QUAY_PASSWORD if the same user can access both namespaces 'eclipse-che-operator-kubernetes' and 'eclipse-che-operator-openshift'"
exit 1
fi
AUTH_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
"user": {
    "username": "'"${QUAY_USERNAME}"'",
    "password": "'"${QUAY_PASSWORD}"'"
}
}' | jq -r '.token')

operator-courier push ${TMP_FLATTEN_DIR} "${QUAY_NAMESPACE}" "${PRJ_NAME}" "${NEW_VERSION}" "${AUTH_TOKEN}"

cd "${CURRENT_DIR}"
