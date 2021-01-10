package conf

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const Project = "github.com/tylerztl/fabric-mempool"

type AppConf struct {
	Conf Application `yaml:"application"`
}

type Application struct {
	Local        bool           `yaml:"local"`
	CPUs         int            `yaml:"cpus"`
	ConnOrderers []*OrdererInfo `yaml:"orderers"`
	OrdererMsp   string         `yaml:"ordererMsp"`
	Channels     []string       `yaml:"channels"`
	TlsEnabled   bool           `yaml:"tlsEnabled"`
	ReqTimeout   int64          `yaml:"reqTimeout"`
}

type OrdererInfo struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

var appConfig = new(AppConf)

func init() {
	confPath := GetConfigPath("app.yaml")
	yamlFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(fmt.Errorf("yamlFile.Get err[%s]", err))
	}
	if err = yaml.Unmarshal(yamlFile, appConfig); err != nil {
		panic(fmt.Errorf("yamlFile.Unmarshal err[%s]", err))
	}
}

func GetAppConf() *AppConf {
	return appConfig
}

// goPath returns the current GOPATH. If the system
// has multiple GOPATHs then the first is used.
func goPath() string {
	gpDefault := build.Default.GOPATH
	gps := filepath.SplitList(gpDefault)

	return gps[0]
}

func GetConfigPath(filename string) string {
	const configPath = "conf"
	return path.Join(goPath(), "src", Project, configPath, filename)
}

func GetCryptoConfigPath(filename string) string {
	return path.Join(goPath(), "src", Project, "sampleconfig/crypto-config", filename)
}

func GetSampleConfigPath() string {
	return path.Join(goPath(), "src", Project, "sampleconfig")
}
