/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var bucketRegion = ""
var prefix = ""
var bucketName = ""
var regexpFilter = ".*"
var dir = "/tmp"
var defrost = false
var download = false

type glacierFile struct {
	key          string
	storageClass string
	size         int64
}

// getFilesCmd represents the getFiles command
var getFilesCmd = &cobra.Command{
	Use:   "getFiles",
	Short: "Retrieve files from glacier by requesting defrost and download if needed",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		getRestoreDownloadObject(bucketRegion, bucketName, prefix, regexpFilter, dir)
	},
}

func init() {
	rootCmd.AddCommand(getFilesCmd)

	getFilesCmd.Flags().StringVarP(&bucketRegion, "bucketRegion", "r", "eu-west-1", "region where the bucket is located")
	getFilesCmd.Flags().StringVarP(&bucketName, "bucketName", "b", "", "name of de bucket to dowload files from")
	getFilesCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "prefix to use to filter result on AWS side ( this increase speed )")
	getFilesCmd.Flags().StringVarP(&regexpFilter, "regexpFilter", "x", ".*", "regular expression used to filter files")
	getFilesCmd.Flags().StringVarP(&dir, "dir", "d", "/tmp", "directory where to save the downloaded files")

	getFilesCmd.Flags().BoolVar(&defrost, "defrost", false, "defrost files that matches")
	getFilesCmd.Flags().BoolVar(&download, "download", false, "download files that matches")

	getFilesCmd.MarkFlagRequired("bucketRegion")
	getFilesCmd.MarkFlagRequired("bucketName")
	getFilesCmd.MarkFlagRequired("regexpFilter")
}

func getRestoreDownloadObject(bucketRegion, bucketName, prefix, regexpFilter, dir string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(bucketRegion)},
	)
	if err != nil {
		exitErrorf("new session error: ", err)
	}

	s3Client := s3.New(sess)

	objectList := getObjects(s3Client, bucketName, prefix, regexpFilter)
	if defrost {
		ok := restoreObjects(s3Client, bucketName, objectList)
		if !ok {
			fmt.Println("Skipping download since some item isn't restored yet")
			return
		}
	}
	if download {
		downloader := s3manager.NewDownloader(sess)
		printSection("Downloading")
		for _, item := range objectList {
			dowloadObject(downloader, bucketName, item, dir)
		}
	}
}

func getObjects(s3Client *s3.S3, bucketName, prefix, regexpFilter string) []glacierFile {
	var glacierFiles []glacierFile

	printSection("Files matching pattern:")

	filter := regexp.MustCompile(regexpFilter)
	pageNum := 0
	err := s3Client.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: &prefix,
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			pageNum++
			for _, object := range page.Contents {
				if filter.MatchString(aws.StringValue(object.Key)) {
					glacierFiles = append(glacierFiles, glacierFile{
						key:          aws.StringValue(object.Key),
						storageClass: aws.StringValue(object.StorageClass),
						size:         aws.Int64Value(object.Size),
					})
					fmt.Println(glacierFiles[len(glacierFiles)-1].storageClass, " - ", glacierFiles[len(glacierFiles)-1].key)
				}
			}
			return !lastPage
		})
	if err != nil {
		exitErrorf("Unable to list items in bucket_name %q, %v", bucketName, err)
	}

	return glacierFiles
}

func restoreObjects(s3Client *s3.S3, bucket string, objectList []glacierFile) (ok bool) {
	ok = true
	printSection("Restoring Objects")
	for _, file := range objectList {
		if file.storageClass == "GLACIER" {

			head := s3.HeadObjectInput{
				Bucket: &bucket,
				Key:    &file.key,
			}
			req, resp := s3Client.HeadObjectRequest(&head)
			err := req.Send()
			if err != nil {
				exitErrorf("could not retrieve file information for %s, error: %v", file.key, err)
			}

			if resp.Restore == nil {
				ok = false
				_, err = s3Client.RestoreObject(
					&s3.RestoreObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(file.key),
						RestoreRequest: &s3.RestoreRequest{
							Days: aws.Int64(1),
						},
					},
				)
				if err != nil {
					exitErrorf("Could not restore %s in bucket %s, %v", file.key, bucket, err)
				}
			} else {
				restoreStatus := strings.Split(aws.StringValue(resp.Restore), ",")[0]
				ongoingRestore := strings.Split(restoreStatus, "=")[1]
				if ongoingRestore == "\"true\"" {
					file.storageClass = "DEFROSTING"
					ok = false
				} else {
					file.storageClass = "DEFROSTED"
				}
			}

			fmt.Println(file.storageClass, " - ", file.key)
		}
	}
	return ok
}

func dowloadObject(downloader *s3manager.Downloader, bucketName string, item glacierFile, dir string) {
	println(dir, item.key)
	os.MkdirAll(dir+filepath.Dir(item.key), os.ModePerm)
	file, err := os.OpenFile(dir+item.key, os.O_CREATE|os.O_WRONLY, os.FileMode(int(0740)))
	if err != nil {
		exitErrorf("Could not open file: ", err)
	}
	defer file.Close()
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(item.key),
		})
	if err != nil {
		exitErrorf("Unable to download item %q, %v", item, err)
	}

	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func printSection(title string) {
	hashes := "#################################################"
	fmt.Printf("\n%s\n## %s\n%s\n", hashes, title, hashes)
}
