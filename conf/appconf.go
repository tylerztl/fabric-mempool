package conf

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const Project = "github.com/tylerztl/fabric-mempool"

type AppConf struct {
	Conf Application `yaml:"application"`
}

type Application struct {
	Local      bool           `yaml:"local"`
	CPUs       int            `yaml:"cpus"`
	Orderers   []*OrdererInfo `yaml:"orderers"`
	TlsEnabled bool           `yaml:"tlsEnabled"`
	ReqTimeout int64          `yaml:"reqTimeout"`
}

type OrdererInfo struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

var appConfig = new(AppConf)

func init() {
	confPath := os.Getenv("MEMPOOL_CONF")
	if confPath == "" {
		confPath = path.Join(goPath(), "src", Project, "conf")
	}

	yamlFile, err := ioutil.ReadFile(path.Join(confPath, "app.yaml"))
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

func GetCryptoConfigPath(filename string) string {
	cfg := os.Getenv("CRYPTO_CONFIG")
	if cfg == "" {
		cfg = path.Join(goPath(), "src", Project, "crypto-config")
	}
	return path.Join(cfg, filename)
}
