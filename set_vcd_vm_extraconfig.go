package main

import (
	"flag"
	"fmt"
	"os"
	"net/url"
	"strings"

	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

type Config struct {
	Host         string
	RefreshToken string
	Insecure     bool
	TenantOrg    string
	TenantVdc    string
	VM           string
	ExtraConfig  extraConfig
}

type extraConfig []*types.ExtraConfigMarshal

func (e *extraConfig) String() string {
	return fmt.Sprintf("%v", *e)
}

func (e *extraConfig) Set(v string) error {
	r := strings.SplitN(v, "=", 2)
	if len(r) < 2 {
		return fmt.Errorf("failed to parse extraConfig: %s", v)
	}
	*e = append(*e, &types.ExtraConfigMarshal{Key: r[0], Value: r[1]})
	return nil
}

func getEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}
	return r
}

func getVCDClient(config *Config) (*govcd.VCDClient, error) {
	u, err := url.ParseRequestURI("https://" + config.Host + "/api")
	vcdclient := govcd.NewVCDClient(*u, config.Insecure)
	err = vcdclient.SetToken(config.TenantOrg, govcd.ApiTokenHeader, config.RefreshToken)
	if err != nil {
		return nil, err
	}
	return vcdclient, nil
}

func main() {

	config := Config{}

	// setup the flags
	flag.StringVar(&config.Host, "url", getEnvString("VCD_URL", ""), "Cloud Director URL [VCD_URL]")
	flag.StringVar(&config.RefreshToken, "token", getEnvString("VCD_TOKEN", ""), "API Token to authenticate to Cloud Director [VCD_TOKEN]")
	flag.BoolVar(&config.Insecure, "insecure", false, "Disable certificate verification")
	flag.StringVar(&config.TenantOrg, "org", "", "Tenant Organization")
	flag.StringVar(&config.TenantVdc, "vdc", "", "Organization Virtual Datacenter")
	flag.StringVar(&config.VM, "vm", "", "Target VM Name")
	flag.Var(&config.ExtraConfig, "e", "ExtraConfig with format <key>=<value>")
	flag.Parse()

	// required arguments
	required := []string{"url", "token", "org", "vdc", "vm", "e"}
	seen := make(map[string]bool)

	// add arguments set with valid default values set to map
	flag.VisitAll(func(f *flag.Flag) {
		if f.DefValue != "[]" && f.DefValue != "" {
			seen[f.Name] = true
		}
	})

	// add arguments explicitly set with a flag to map
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	// check if any required arguments are missing
	for _, req := range required {
		if !seen[req] {
			flag.Usage()
			os.Exit(2)
		}
	}

	// connect to cloud director
	client, err := getVCDClient(&config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// query the client for the org object
	org, err := client.GetOrgByName(config.TenantOrg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// get the vdc object
	vdc, err := org.GetVDCByName(config.TenantVdc, false)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// get the vm object
	vm, err := vdc.QueryVmByName(config.VM)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// set extraconfig properties on vm
	_, err = vm.UpdateExtraConfig(config.ExtraConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}