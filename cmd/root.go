package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"

	"github.com/AlecAivazis/survey/v2"
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

func loadConfigAndSave(cfg *config.Config, safe bool) {
	cfgYAML, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	err = viper.ReadConfig(bytes.NewBuffer(cfgYAML))
	if err != nil {
		log.Fatalf("Unable to create a default configuration : %v", err)
	}
	if safe {
		err = viper.SafeWriteConfig()
	} else {
		err = viper.WriteConfig()
	}
	if err != nil {
		log.Fatalf("Unable to save default configuration file : %v", err)
	}
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
			loadConfigAndSave(&defaultCfg, true)
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

func runInteractiveSetup(cfg *config.Config) error {
	fmt.Println("Begining interactive setup")
	questions := []*survey.Question{
		{
			Name: "port",
			Prompt: &survey.Input{
				Message: "Choose a defaut port (can be overridden later by command-line arguments)",
				Default: "27010",
			},
			Validate: func(val interface{}) error {
				errRet := errors.New("The response should be an integer between 1 and 65535")
				str, ok := val.(string)
				if !ok {
					return errRet
				}
				x, err := strconv.Atoi(str)
				if err != nil || x <= 0 || x > 65535 {
					return errRet
				}
				return nil
			},
		},
		{
			Name: "domain",
			Prompt: &survey.Input{
				Message: "Choose a domain name to bind with the server",
				Default: "localhost",
			},
			Validate: survey.MinLength(3),
		},
		{
			Name: "database_url",
			Prompt: &survey.Input{
				Message: "Enter the URL of the MongoDB database",
				Help:    "The URL should follows this template : 'mongodb+srv://<user>:<password>@<hostname>[:<port>]/<database>'",
			},
			Validate: survey.Required,
		},
		{
			Name: "database_name",
			Prompt: &survey.Input{
				Message: "Enter the name of the MongoDB database",
			},
			Validate: survey.Required,
		},
	}
	answers := struct {
		Port         uint16
		Domain       string
		DatabaseURL  string `survey:"database_url"`
		DatabaseName string `survey:"database_name"`
	}{}
	err := survey.Ask(questions, &answers)
	if err != nil {
		if err != terminal.InterruptErr {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}

	cfg.Port = answers.Port
	cfg.Domain = answers.Domain
	cfg.Database.URL = answers.DatabaseURL
	cfg.Database.Name = answers.DatabaseName

	fmt.Println("End of interactive setup, please launch the application again.")

	return nil
}

func runRootCmd(cmd *cobra.Command, args []string) {
	if !mainCfg.IsValid() {
		fmt.Println("No valid configuration found...")
		err := runInteractiveSetup(&mainCfg)
		if err != nil {
			fmt.Printf("An error has occured during the interactive setup: %v", err)
		}
		loadConfigAndSave(&mainCfg, false)
		return
	}
	masterServer = server.NewMasterServer(mainCfg)
	if err := masterServer.Listen(); err != nil {
		log.Fatalf("An error has occured with the server: %v", err)
	}
}
