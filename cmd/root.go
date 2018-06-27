// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/paulczar/gcp-lb-tags/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	requiredFlags = []string{"name", "project", "network"}
	config        = &util.Config{}
)

const ()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gcp-lb-tags",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gcp-lb-tags.yaml)")

	rootCmd.PersistentFlags().StringVarP(&config.Region, "region", "r", "us-central1", "GCP region")
	rootCmd.PersistentFlags().StringVarP(&config.ProjectID, "project", "p", "", "Project ID")
	rootCmd.PersistentFlags().StringVarP(&config.Name, "name", "n", "", "Name of loadbalancer")
	rootCmd.PersistentFlags().StringVar(&config.Network, "network", "", "GCP network")
	rootCmd.PersistentFlags().StringSliceP("tags", "t", []string{}, "Tags to Load Balance for")
	rootCmd.PersistentFlags().StringSliceP("labels", "l", []string{}, "Labels to Load Balance for")
	rootCmd.PersistentFlags().StringVar(&config.Address, "address", "", "Name of the external IP address to attach to LB if different to LB name")
	rootCmd.PersistentFlags().StringSlice("ports", []string{"8443"}, "Ports to load balance for")
	rootCmd.PersistentFlags().StringSliceP("zones", "z", []string{"a", "b", "c"}, "zones your compute instances are in (will be appended to value of --region")
	for _, f := range requiredFlags {
		rootCmd.MarkPersistentFlagRequired(f)
	}

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gcp-lb-tags" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gcp-lb-tags")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
