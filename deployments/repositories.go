package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"

	"path"
)

//This function allow you to retrieve the url of a repo from is name
func GetRepositoryUrlFromName(kv *api.KV, deploymentID, repoName string) (err error, url string) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "url"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occured when trying to get repositoty url")
		return
	}

	if res == nil {
		err = errors.Errorf("The repository %v has been not found", repoName)
		return
	}

	url = string(res.Value)
	return
}
