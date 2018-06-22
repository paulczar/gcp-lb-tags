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
	"github.com/spf13/cobra"
)

var loop bool

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
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
		config.Zones = util.ExpandZones(config, util.GetFlagStringSlice(cmd, "zones"))
		fmt.Printf("Ensuring that TargetPool %s contains instances in %s with %v\n", config.Name, config.Region, config.Tags)
		if loop {
			for {
				util.AddorDelInstances(config)
				time.Sleep(120 * time.Second)
			}
		} else {
			util.AddorDelInstances(config)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	// Here you will define your flags and configuration settings.
	runCmd.Flags().BoolVar(&loop, "loop", false, "run in a continuous (120sec) loop")
}
