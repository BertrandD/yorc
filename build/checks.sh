#!/usr/bin/env bash
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#      http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#set -x
scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

get_consul_version () {
    grep consul_version "${scriptDir}/../versions.yaml" | awk '{print $2}'
}

error_exit () {
    >&2 echo "${1}"
    if [[ $# -gt 1 ]]
    then
        exit ${2}
    else
        exit 1
    fi
}

install_consul() {
    cd ${scriptDir}
    consulVersion=$(get_consul_version)
    zipName="consul_${consulVersion}_$(go env GOHOSTOS)_$(go env GOHOSTARCH).zip"
    wget "https://releases.hashicorp.com/consul/${consulVersion}/${zipName}"
    unzip ${zipName}
    rm ${zipName}
    chmod +x consul
}


if [[ -z "$GOROOT" ]]; then
    error_exit "GOROOT env var should be set..."
fi

if [[ -z "$GOPATH" ]]; then
    error_exit "GOPATH env var should be set..."
fi

for tool in $@; do
    #Suppress trailing /... in url if any
    tool="${tool%%/...*}"
    if [[ ! -x $GOPATH/bin/${tool##*/} ]]; then
        error_exit "Tool not found $GOPATH/bin/${tool##*/} doesn't exist. This could be fixed by running 'make tools'"
    fi
done

if [[ ! -x "${scriptDir}/consul" ]]; then
    rm -f "${scriptDir}/consul"
    install_consul
else
    installedConsulVersion=$(${scriptDir}/consul version | grep "Consul v" | cut -d 'v' -f2)
    consulVersion=$(get_consul_version)
    if [[ "${installedConsulVersion}" != "${consulVersion}" ]]; then
        rm -f "${scriptDir}/consul"
        install_consul
    fi
fi
