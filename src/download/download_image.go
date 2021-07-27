package download

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/config"
	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/image"
	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/log"
	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/registry"
)

func downloadImage(images []string) {

	var r *registry.Registry
	var err error
	rs := registry.NewRegistryStore()

	saveDir := path.Join(config.Kconfig.DownloadDir, "download_images")
	os.MkdirAll(saveDir, 0755)

	download := func(repoTag imageRepo) {
		log.Log.WithField("domain", repoTag.domain).WithField("repo", repoTag.repo).WithField("tag", repoTag.tag).Debug()
		if repoTag.domain == "" || repoTag.repo == "" || repoTag.tag == "" {
			return
		}

		if r, err = rs.Get(repoTag.domain); err != nil {
			if config.Kconfig.RegistryUser == "" || config.Kconfig.RegistryPassword == "" {
				//不需要用户名密码
				r, err = registry.NewRegistry(registry.Config{Endpoint: repoTag.domain})
			} else {
				r, err = registry.NewRegistry(registry.Config{Endpoint: repoTag.domain, Username: config.Kconfig.RegistryUser, Password: config.Kconfig.RegistryPassword})
			}
			if err != nil {
				log.Log.Fatal(err.Error())
			}
			rs.Add(repoTag.domain, r)
		}

		tarFile := fmt.Sprintf("%s_%s.tar.gz", path.Join(saveDir, strings.Replace(repoTag.repo, "/", "_", 1)), repoTag.tag)
		if _, ok := os.Stat(tarFile); ok == nil {
			log.Log.Info(tarFile, " is exist. skip download.")
			return
		}
		log.Log.Info("downloading ", repoTag.domain, "/", repoTag.repo, ":", repoTag.tag)
		mF, err := r.ReposManifests(repoTag.repo, repoTag.tag)
		if err != nil {
			log.Log.Fatal(err.Error())
		}
		dir, err := ioutil.TempDir("", "graboid")
		if err != nil {
			log.Log.Fatal(err.Error())
		}
		defer os.RemoveAll(dir) // clean up

		cfile, err := r.RepoGetConfig(dir, repoTag.repo, mF)
		if err != nil {
			log.Log.Fatal(err.Error())
		}

		//log.Infof(getFmtStr(), "GET LAYERS")
		lfiles, err := r.RepoGetLayers(dir, repoTag.repo, mF)
		if err != nil {
			log.Log.Fatal(err.Error())
		}

		//log.Infof(getFmtStr(), "CREATE manifest.json")
		_, err = createManifest(dir, cfile, repoTag.imageRepoTags, lfiles)
		if err != nil {
			log.Log.Fatal(err.Error())
		}
		//create dir

		if runtime.GOOS == "windows" {
			log.Log.Infof("%s: %s", "CREATE docker image tarball", tarFile)
		} else {
			log.Log.Infof("\033[1m%s:\033[0m \033[34m%s\033[0m", "CREATE docker image tarball", tarFile)
		}
		err = tarFiles(dir, tarFile)
		if err != nil {
			log.Log.Fatal(err.Error())
		}
		log.Log.Infof("\033[1mSUCCESS!\033[0m")
	}
	for _, v := range images {
		repoTag := newImageRepo(v)
		download(repoTag)
	}
}

func createManifest(tempDir, confFile, repoTag string, layerFiles []string) (string, error) {
	var manifestArray []image.Manifest
	// Create the file
	tmpfn := filepath.Join(tempDir, "manifest.json")
	out, err := os.Create(tmpfn)
	if err != nil {
		log.Log.WithError(err).Error("create manifest JSON failed")
	}
	defer out.Close()

	//tag := getTag(repoTag)
	_, repo, tag := decomposeRepoTag(repoTag)
	if tag == "" {
		tag = "latest"
	}
	m := image.Manifest{
		Config:   confFile,
		Layers:   layerFiles,
		RepoTags: []string{strings.Join([]string{repo, tag}, ":")}, //更改镜像repo
	}
	manifestArray = append(manifestArray, m)
	mJSON, err := json.Marshal(manifestArray)
	if err != nil {
		log.Log.WithError(err).Error("marshalling manifest JSON failed")
	}
	// Write the body to JSON file
	_, err = out.Write(mJSON)
	if err != nil {
		log.Log.WithError(err).Error("writing manifest JSON failed")
	}

	return tmpfn, nil
}
func tarFiles(srcDir, tarName string) error {
	tarfile, err := os.Create(tarName)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	gw := gzip.NewWriter(tarfile)
	defer gw.Close()
	tarball := tar.NewWriter(gw)
	defer tarball.Close()

	return filepath.Walk(srcDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			if err = tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			log.Log.WithField("path", path).Debug("taring file")
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}
