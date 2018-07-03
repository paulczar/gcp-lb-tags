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
	"time"

	"github.com/paulczar/gcp-lb-tags/pkg"
	"github.com/paulczar/gcp-lb-tags/pkg/cloud"
	"github.com/spf13/cobra"
)

var (
	loop    bool
	seconds int
)

// createCmd represents the run command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "updates GCP load balancer based on instance tags",
	Long: `
gcp-lb-tags accepts a list of tags and will monitor a named load balancer's target
pool and will modify that target pool to match the list of compute instances that
match that those tags.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		config.Tags = util.GetFlagStringSlice(cmd, "tags")
		config.Labels = util.GetFlagStringSlice(cmd, "labels")
		if config.Address == "" {
			config.Address = config.Name
		}
		client, err := cloud.New(config.ProjectID, config.Network, config.Region)
		if err != nil {
			panic(err)
		}
		config.Zones = nil
		if loop {
			for {
				err := client.CreateLoadBalancer(config)
				if err != nil {
					fmt.Printf("An error Occured: %s\n\n\n", err)
				}
				time.Sleep(time.Duration(seconds) * time.Second)
			}
		} else {
			//fmt.Printf("zones: %v", config.Zones)
			return client.CreateLoadBalancer(config)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	// Here you will define your flags and configuration settings.
	createCmd.Flags().BoolVar(&loop, "loop", false, "run in a continuous [seconds] loop")
	createCmd.Flags().IntVar(&seconds, "seconds", 120, "how long between each loop in seconds")
}
