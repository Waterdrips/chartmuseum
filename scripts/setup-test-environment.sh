#!/bin/bash -ex

HELM_VERSION="2.17.0"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

export PATH="$PWD/testbin:$PATH"

main() {
    export HELM_HOME="$PWD/.helm"
    package_test_charts
}

package_test_charts() {
    pushd testdata/charts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        helm package --sign --key helm-test --keyring ../../pgp/helm-test-key.secret .
        popd
    done
    # add another version to repo for metric tests
    helm package --sign --key helm-test --keyring ../pgp/helm-test-key.secret --version 0.2.0 -d mychart/ mychart/.
    popd

    pushd testdata/badcharts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        # TODO: remove in v0.14.0. We do not generate .prov file for this chart
        # since prov validation is not enabled, and it breaks acceptance tests
        if grep "mybadsemver2chart" Chart.yaml; then
            helm package . || true
        fi
        popd
    done
}

main
