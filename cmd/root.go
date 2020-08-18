package cmd

import (
	"bytes"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jbltx/master-server/config"
	"github.com/jbltx/master-server/server"
)

var (
	port         uint16
	cfgFile      string
	masterServer *server.MasterServer
	mainCfg      config.Config
)

// RootCmd is the root command used to start the Master Server
var RootCmd = &cobra.Command{
	Short: "Master Server - v0.1.0",
	Run:   runRootCmd,
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().Uint16VarP(&port, "port", "p", 27010, "The port to bind to")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config-file", "c", "", "config file (default is $HOME/.master-server/config.yaml)")
}

func initConfig() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if cfgFile != "" {
		_, err = os.Stat(cfgFile)
		if err == nil {
			viper.SetConfigFile(cfgFile)
			configDir := path.Dir(cfgFile)
			viper.AddConfigPath(configDir)
		} else {
			if os.IsNotExist(err) {
				log.Fatalf("Unable to find a configuration file at %s. Aborting process.\n", cfgFile)
			} else {
				log.Fatalf("An error has occured during config file path info: %v", err)
			}
		}
	}

	homeDir, err := homedir.Dir()
	if err != nil {
		log.Printf("Unable to find home directory for configuration file path: %v", err)
	} else {
		homeCfgPath := path.Join(homeDir, ".master-server")
		os.MkdirAll(homeCfgPath, os.ModeDir)
		viper.AddConfigPath(homeCfgPath)
	}
	viper.AddConfigPath(dir)

	viper.SetEnvPrefix("MASTERSERVER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			defaultCfg := config.NewDefaultConfig()
			cfgYAML, err := yaml.Marshal(defaultCfg)
			if err != nil {
				panic(err)
			}
			err = viper.ReadConfig(bytes.NewBuffer(cfgYAML))
			if err != nil {
				log.Fatalf("Unable to create a default configuration : %v", err)
			}
			err = viper.SafeWriteConfig()
			if err != nil {
				log.Fatalf("Unable to save default configuration file : %v", err)
			}
		} else {
			log.Fatal(err)
		}
	}
	// ? (jbltx) Do we need to watch config file for changes ?

	err = viper.Unmarshal(&mainCfg)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
}

func runInteractiveSetup() {
	// todo (jbltx) create interactive setup here
}

func runRootCmd(cmd *cobra.Command, args []string) {
	if !mainCfg.IsValid() {
		runInteractiveSetup()
	}
	masterServer = server.NewMasterServer(mainCfg)
	if err := masterServer.Listen(); err != nil {
		log.Fatalf("An error has occured with the server: %v", err)
	}
}
