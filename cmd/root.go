/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rgolangh/ovirt-image-upload/pkg/upload"
)

var (
	src             string
	storageDomainId string
)

// imageUploadCmd represents the imageUpload command
var imageUploadCmd = &cobra.Command{
	Use:   "image-upload",
	Short: "stream an image from a URL or file to an oVirt storage domain.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := upload.Upload(src, storageDomainId)
		if err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	imageUploadCmd.Flags().StringVarP(
		&src, "src", "s", "", "A URL of an image file to upload to. Can be https://a/b/c or file:///a/b/c")
	imageUploadCmd.Flags().StringVarP(
		&storageDomainId, "storage-domain-id", "d", "", "The Storage Domain ID to create the disk in.")

	err := imageUploadCmd.MarkFlagRequired("src")
	if err != nil {
		logrus.Fatal(err)
	}
	err = imageUploadCmd.MarkFlagRequired("storage-domain-id")
	if err != nil {
		logrus.Fatal(err)
	}
}

func Execute() {
	imageUploadCmd.Execute()
}
