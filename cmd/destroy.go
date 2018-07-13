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
	"github.com/paulczar/gcp-lb-tags/pkg/cloud"
	"github.com/spf13/cobra"
)

var (
	force bool
)

// removeCmd represents the run command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "removes GCP load balancer based on instance tags",
	Long: `
gcp-lb-tags accepts a list of tags and will monitor a named load balancer's target
pool and will modify that target pool to match the list of compute instances that
match that those tags.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		//		config.Tags = util.GetFlagStringSlice(cmd, "tags")
		//		config.Labels = util.GetFlagStringSlice(cmd, "labels")
		if config.Address == "" {
			config.Address = config.Name
		}
		client, err := cloud.New(config.ProjectID, config.Network, config.Region)
		if err != nil {
			return err
		}
		return client.RemoveLoadBalancer(config, force)
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	// Here you will define your flags and configuration settings.
	destroyCmd.Flags().BoolVar(&force, "force", false, "Force deletion of all objects (by default does not delete external IP).")
}
