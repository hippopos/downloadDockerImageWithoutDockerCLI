package download

import (
	"strings"

	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/log"
)

type imageRepo struct {
	imageRepoTags string
	tag           string
	domain        string //registry-1.docker.io
	repo          string //library/busybox
}

//分解image Repo 为 domain 和 repo 和tag;
//比如 busybox解析为 registry-1.docker.io  library/busybox latest
//bitnami/rabbitmq 解析为 registry-1.docker.io  bitnami/rabbitmq latest
//registry.cn-zhangjiakou.aliyuncs.com/bizconf_devops/bizconf_redis_manager 解析为 registry.cn-zhangjiakou.aliyuncs.com  bizconf_devops/bizconf_redis_manager latest
func newImageRepo(imageRepoTags string) (image imageRepo) {

	if imageRepoTags == "" {
		return image
	}
	image.imageRepoTags = imageRepoTags
	var domain, repo, tag string

	domain,repo,tag = decomposeRepoTag(imageRepoTags)
	//dockerhub
	if domain == ""{
		domain = "registry-1.docker.io"
		//append library
		if !strings.Contains(repo, "/") {
			repo = strings.Join([]string{"library", repo}, "/")
		}
	}
	//set latest
	if tag == ""{
		tag = "latest"
	}

	image.tag = tag
	image.repo = repo
	image.domain = domain
	log.Log.Debug("Before: ", imageRepoTags, " After: ", strings.Join([]string{strings.Join([]string{domain, repo}, "/"), tag}, ":"))
	return image
}

func decomposeRepoTag(repoTag string) (domain, repo, tag string) {
	// busybox
	// localhost/busybox
	// busybox:alphine
	// xxxx:5000/busybox
	// xxxx:5000/lib/busybox:alphine

	//含有 :
	if strings.Contains(repoTag, ":") {
		//分解 :
		strs := strings.Split(repoTag, ":")
		last := strs[len(strs)-1]

		//最后string:含有 /,说明没有tag 只有domain repo
		if strings.Contains(last, "/") {
			//劈到了:5000/xxx
			tag = ""
			domain = strings.Split(repoTag, "/")[0]
			for i, s := range strings.Split(repoTag, "/") {
				if i != 0 {
					if i == 1 {
						repo = s
					} else {
						repo = strings.Join([]string{repo, s}, "/")
					}
				}
			}
		} else {
			//最后string 没有 /,说明是tag
			//劈到了:tag
			tag = last
			//去掉tag
			repoTag = strings.TrimSuffix(repoTag, ":"+tag)

			domain = strings.Split(repoTag, "/")[0]
			if !strings.Contains(domain, ".") && domain != "localhost" && !strings.Contains(domain, ":") {
				//缺省domain
				domain = ""
				repo = repoTag
			} else {
				//存在domain
				for i, s := range strings.Split(repoTag, "/") {
					if i != 0 {
						if i == 1 {
							repo = s
						} else {
							repo = strings.Join([]string{repo, s}, "/")
						}
					}
				}
			}
		}

	} else {
		//缺省 :
		domain = strings.Split(repoTag, "/")[0]
		if !strings.Contains(domain, ".")&& domain != "localhost" && !strings.Contains(domain, ":") {
			//缺省domain
			domain = ""
			repo = repoTag
		} else {
			//存在domain
			for i, s := range strings.Split(repoTag, "/") {
				if i != 0 {
					if i == 1 {
						repo = s
					} else {
						repo = strings.Join([]string{repo, s}, "/")
					}
				}
			}
		}
		tag = ""
	}

	return domain, repo, tag
}
