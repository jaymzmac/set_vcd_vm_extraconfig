package set_vcd_vm_extraconfig

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vmware/cloud-provider-for-cloud-director/pkg/vcdsdk"
)

type Config struct {
	Host         string
	User         string
	RefreshToken string
	Insecure     bool
	TenantOrg    string
	TenantVdc    string
	VM           string
	ExtraConfig  extraConfig
	PowerOn      bool
}

type extraConfig []BaseOptionValue

type BaseOptionValue interface {
	GetOptionValue() *OptionValue
}

type OptionValue struct {
	DynamicData
	Key   string  `xml:"key" json:"key"`
	Value AnyType `xml:"value,typeattr" json:"value"`
}

type DynamicData struct{}

type AnyType interface{}

func (b *OptionValue) GetOptionValue() *OptionValue { return b }

func (e *extraConfig) String() string {
	return fmt.Sprintf("%v", *e)
}

func (e *extraConfig) Set(v string) error {
	r := strings.SplitN(v, "=", 2)
	if len(r) < 2 {
		return fmt.Errorf("failed to parse extraConfig: %s", v)
	}
	*e = append(*e, &OptionValue{Key: r[0], Value: r[1]})
	return nil
}

func getEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	return r
}

func getVCDClient(config *Config) (*vcdsdk.Client, error) {
	password := ""
	getVdcClient := true

	return vcdsdk.NewVCDClientFromSecrets(
		config.Host,
		config.TenantOrg,
		config.TenantVdc,
		config.TenantOrg,
		config.User,
		password,
		config.RefreshToken,
		config.Insecure,
		getVdcClient,
	)
}

func main() {

	config := Config{}

	// setup the flags
	flag.StringVar(&config.Host, "url", getEnvString("VCD_URL", ""), "Cloud Director URL [VCD_URL]")
	flag.StringVar(&config.User, "user", getEnvString("VCD_USER", ""), "User to connect to Cloud Director [VCD_USER]")
	flag.StringVar(&config.RefreshToken, "token", getEnvString("VCD_TOKEN", ""), "API Token to authenticate to Cloud Director [VCD_TOKEN]")
	flag.BoolVar(&config.Insecure, "insecure", false, "Disable certificate verification")
	flag.StringVar(&config.TenantOrg, "org", "", "Tenant Organization")
	flag.StringVar(&config.TenantVdc, "vdc", "", "Organization Virtual Datacenter")
	flag.StringVar(&config.VM, "vm", "", "Target VM Name")
	flag.BoolVar(&config.PowerOn, "poweron", false, "Power On VM after setting ExtraConfig")
	flag.Var(&config.ExtraConfig, "e", "ExtraConfig with format <key>=<value>")
	flag.Parse()

	// required arguments
	required := []string{"url", "user", "token", "org", "vdc", "vm", "e"}
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

	// query the client for the vm object
	vm, err := client.VDC.QueryVmByName(config.VM)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create vdcmanager
	vdcManager, err := vcdsdk.NewVDCManager(client, client.ClusterOrgName, client.ClusterOVDCName)

	// set extraconfig properties on vm
	for _, item := range config.ExtraConfig {
		err = vdcManager.SetVmExtraConfigKeyValue(vm, item.GetOptionValue().Key, item.GetOptionValue().Value.(string), false)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// Power On VM
	if config.PowerOn {
		vmStatus, err := vm.GetStatus()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if vmStatus != "POWERED_ON" {
			task, err := vm.PowerOn()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if err = task.WaitTaskCompletion(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}

}
