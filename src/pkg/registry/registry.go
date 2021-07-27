package registry

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/log"
)

const (
	Version = "v2"
)

type octetType byte

var octetTypes [256]octetType

const (
	isToken octetType = 1 << iota
	isSpace
)

type Access struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Actions []string `json:"actions"`
}
type authPayloadJWT struct {
	Iss    string   `json:"iss"`
	Aud    string   `json:"aud"`
	Iat    int64    `json:"iat"`
	Exp    int64    `json:"exp"`
	Access []Access `json:"access"`
}

// Config registry config struct
type Config struct {
	Endpoint       string //https://registry-1.docker.io
	RegistryDomain string
	Proxy          string
	Insecure       bool
	Username       string
	Password       string
}

// Registry registry object
type Registry struct {
	URL          *url.URL
	Host         string
	RegistryHost string
	client       *http.Client
	Auth         auth
	Config       Config
	Realm        string //https://dockerauth-cn-zhangjiakou.aliyuncs.com/auth
	Service      string //registry.aliyuncs.com:cn-zhangjiakou:78456
}

type auth struct {
	Token       string    `json:"token,omitempty"`
	AccessToken string    `json:"access_token,omitempty"`
	ExpiresIn   int       `json:"expires_in,omitempty"`
	IssuedAt    time.Time `json:"issued_at,omitempty"`
}

// Tags is the image tags struct
type Tags struct {
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// Manifests is the image manifest struct
type Manifests struct {
	Config        manifestConfig  `json:"config,omitempty"`
	Layers        []manifestLayer `json:"layers,omitempty"`
	MediaType     string          `json:"mediaType,omitempty"`
	SchemaVersion int             `json:"schemaVersion,omitempty"`
}

type manifestConfig struct {
	Digest    string `json:"digest,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Size      int    `json:"size,omitempty"`
}

type manifestLayer struct {
	Digest    string `json:"digest,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Size      int    `json:"size,omitempty"`
}

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <">
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := 0; c < 256; c++ {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.IndexRune(" \t\"(),/:;<=>?@[]\\{}", rune(c)) >= 0
		if strings.IndexRune(" \t\r\n", rune(c)) >= 0 {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}

func getProxy(proxy string) func(*http.Request) (*url.URL, error) {
	if len(proxy) > 0 {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			log.Log.WithError(err).Error("bad proxy url")
		}
		log.Log.Debugf("proxy set to: %s", proxyURL)
		return http.ProxyURL(proxyURL)
	}
	conf := httpproxy.FromEnvironment()
	log.Log.WithFields(logrus.Fields{
		"http_proxy":  conf.HTTPProxy,
		"https_proxy": conf.HTTPSProxy,
		"no_proxy":    conf.NoProxy,
	}).Debugf("proxy info from environment")
	return http.ProxyFromEnvironment
}

func parse(p string) ([]byte, error) {
	parts := strings.Split(p, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("oidc: malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("oidc: malformed jwt payload: %v", err)
	}
	return payload, nil
}

func (reg *Registry) TokenExpired(repoName string) bool {
	//每个repo都需要重新获取token

	if reg.Auth.Token == "" {
		return true
	}
	jwtpayload, err := parse(reg.Auth.Token)
	if err != nil {
		log.Log.Error(err.Error())
		return true
	}
	var p authPayloadJWT
	err = json.Unmarshal(jwtpayload, &p)
	if err != nil {
		log.Log.Error(err.Error())
		return true
	}

	duration := time.Since(time.Unix(p.Exp, 0))
	if int(duration.Seconds()) > 0 {
		return true
	}
	//如果token access.name记录的是当前repo,则不过期
	for _, v := range p.Access {
		if v.Name == repoName {
			return false
		}
	}
	return true
}

// New creates a new Registry object
func NewRegistry(rc Config) (*Registry, error) {
	u, e := url.Parse(rc.Endpoint)
	if e != nil {
		return nil, e
	}
	if u.Host == "" {
		u.Host = u.Path
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	origURL := u
	host := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	registryHost := host
	if rc.RegistryDomain != "" {
		u, e = url.Parse(rc.RegistryDomain)
		if e != nil {
			return nil, e
		}
		if u.Host == "" {
			u.Host = u.Path
		}
		registryHost = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	} else {
		rc.RegistryDomain = u.Host
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           getProxy(rc.Proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: rc.Insecure},
		},
	}

	r := &Registry{
		URL:          origURL,
		Host:         host,
		RegistryHost: registryHost,
		client:       client,
		Config:       rc}
	err := r.Dial()
	return r, err
}

func (reg *Registry) Dial() error {
	//无认证访问 http://registry.cn-zhangjiakou.aliyuncs.com/v2  返回 401, 并且header携带auth server地址

	urlStr := fmt.Sprintf("https://%s/%s", reg.Config.RegistryDomain, Version)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	res, err := reg.client.Do(req)
	if err != nil {
		return err
	}
	for _, v := range parseAuthHeader(res.Header) {
		if v.Scheme == "bearer" {
			reg.Realm, _ = v.Parameters["realm"]
			reg.Service, _ = v.Parameters["service"]
		}
	}
	return nil
}

// GetToken retrives a docker registry API pull token
func (reg *Registry) GetToken(repoName string) error {
	//reg.Auth.Token = "ZG9ja2VycmVwb0BiaXpjb25mOmFkbWluMTIz"
	//return nil
	//u := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", reg.Config.RepoName)
	u := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", reg.Realm, reg.Service, repoName)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	if reg.Config.Username != "" && reg.Config.Password != "" {
		req.SetBasicAuth(reg.Config.Username, reg.Config.Password)
	}
	res, err := reg.client.Do(req)
	if err != nil {
		log.Log.Debug("Get token: url: %s; header: %+v", u, req.Header)
		log.Log.Error("Get token: ", err.Error())
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("HTTP Error: %s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Log.Error("Get token: ", err.Error())
		return err
	}

	var a = new(auth)
	err = json.Unmarshal(body, &a)
	if err != nil {
		return err
	}

	reg.Auth = *a
	log.Log.WithField("token", a.Token).Debugf("got token")

	return nil
}

func (reg *Registry) doGet(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", reg.Auth.Token))
	req.Header.Add("User-Agent", "Docker-Client/18.06.0-ce (darwin)")
	// add additional headers
	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}
	res, err := reg.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP Error: %s", res.Status)
	}
	return res, nil
}

// ReposTags gets a list of the docker image tags
func (reg *Registry) ReposTags(reposName string) (*Tags, error) {
	//url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/tags/list", reposName)
	url := fmt.Sprintf("%s/v2/%s/tags/list", reg.Host, reposName)

	if reg.TokenExpired(reposName) {
		reg.GetToken(reposName)
	}

	log.Log.WithFields(logrus.Fields{
		"url": url,
	}).Debug("downloading tags")
	res, err := reg.doGet(url, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	t := new(Tags)
	if err := json.Unmarshal(rawJSON, &t); err != nil {
		return nil, err
	}

	return t, nil
}

// ReposManifests gets docker image manifest for name:tag
func (reg *Registry) ReposManifests(reposName, repoTag string) (*Manifests, error) {
	headers := make(map[string]string)
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", reg.Host, reposName, repoTag)
	headers["Accept"] = "application/vnd.docker.distribution.manifest.v2+json"
	log.Log.WithFields(logrus.Fields{
		"url":     url,
		"headers": headers,
		"image":   reposName,
		"tag":     repoTag,
	}).Debug("get manifests")

	if reg.TokenExpired(reposName) {
		reg.GetToken(reposName)
	}

	res, err := reg.doGet(url, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	m := new(Manifests)
	if err := json.Unmarshal(rawJSON, &m); err != nil {
		return nil, err
	}

	return m, nil
}

// RepoGetConfig gets docker image config JSON
func (reg *Registry) RepoGetConfig(tempDir, reposName string, manifest *Manifests) (string, error) {
	// Create the file
	tmpfn := filepath.Join(tempDir, fmt.Sprintf("%s.json", strings.TrimPrefix(manifest.Config.Digest, "sha256:")))
	out, err := os.Create(tmpfn)
	if err != nil {
		log.Log.WithError(err).Error("create config file failed")
	}
	defer out.Close()
	// Download config
	headers := make(map[string]string)
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", reg.Host, reposName, manifest.Config.Digest)
	headers["Accept"] = manifest.Config.MediaType
	log.Log.WithField("url", url).Debug("downloading config")

	if reg.TokenExpired(reposName) {
		reg.GetToken(reposName)
	}

	res, err := reg.doGet(url, headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, res.Body)
	if err != nil {
		log.Log.WithError(err).Error("writing config file failed")
	}

	return filepath.Base(tmpfn), nil
}

// RepoGetLayers gets docker image layer tarballs
func (reg *Registry) RepoGetLayers(tempDir, reposName string, manifest *Manifests) ([]string, error) {
	var layerFiles []string

	for _, layer := range manifest.Layers {
		// Create the TAR file
		tmpfn := filepath.Join(tempDir, fmt.Sprintf("%s.tar", strings.TrimPrefix(layer.Digest, "sha256:")))
		layerFiles = append(layerFiles, filepath.Base(tmpfn))
		out, err := os.Create(tmpfn)
		if err != nil {
			log.Log.WithError(err).Error("create tar file failed")
		}
		defer out.Close()

		// Download layer
		headers := make(map[string]string)
		url := fmt.Sprintf("%s/v2/%s/blobs/%s", reg.Host, reposName, layer.Digest)
		headers["Accept"] = layer.MediaType
		log.Log.WithField("url", url).Debug("downloading layer")

		if reg.TokenExpired(reposName) {
			reg.GetToken(reposName)
		}

		res, err := reg.doGet(url, headers)
		if err != nil {
			log.Log.Error(err.Error())
			return nil, err
		}
		defer res.Body.Close()
		// create progressbar
		bar := pb.New(layer.Size).SetUnits(pb.U_BYTES)
		bar.SetWidth(90)
		bar.Start()
		reader := bar.NewProxyReader(res.Body)
		// Write the body to file
		_, err = io.Copy(out, reader)
		if err != nil {
			log.Log.Error("Get layers: writing tar file failed")
			return nil, err
		}
		bar.Finish()
	}

	return layerFiles, nil
}
func parseAuthHeader(header http.Header) []challenge.Challenge {
	challenges := []challenge.Challenge{}
	for _, h := range header[http.CanonicalHeaderKey("WWW-Authenticate")] {
		v, p := parseValueAndParams(h)
		if v != "" {
			challenges = append(challenges, challenge.Challenge{Scheme: v, Parameters: p})
		}
	}
	return challenges
}
func parseValueAndParams(header string) (value string, params map[string]string) {
	params = make(map[string]string)
	value, s := expectToken(header)
	if value == "" {
		return
	}
	value = strings.ToLower(value)
	s = "," + skipSpace(s)
	for strings.HasPrefix(s, ",") {
		var pkey string
		pkey, s = expectToken(skipSpace(s[1:]))
		if pkey == "" {
			return
		}
		if !strings.HasPrefix(s, "=") {
			return
		}
		var pvalue string
		pvalue, s = expectTokenOrQuoted(s[1:])
		if pvalue == "" {
			return
		}
		pkey = strings.ToLower(pkey)
		params[pkey] = pvalue
		s = skipSpace(s)
	}
	return
}
func expectToken(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isToken == 0 {
			break
		}
	}
	return s[:i], s[i:]
}

func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

func expectTokenOrQuoted(s string) (value string, rest string) {
	if !strings.HasPrefix(s, "\"") {
		return expectToken(s)
	}
	s = s[1:]
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			return s[:i], s[i+1:]
		case '\\':
			p := make([]byte, len(s)-1)
			j := copy(p, s[:i])
			escape := true
			for i = i + 1; i < len(s); i++ {
				b := s[i]
				switch {
				case escape:
					escape = false
					p[j] = b
					j++
				case b == '\\':
					escape = true
				case b == '"':
					return string(p[:j]), s[i+1:]
				default:
					p[j] = b
					j++
				}
			}
			return "", ""
		}
	}
	return "", ""
}
