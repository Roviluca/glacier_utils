# Glacier_utils

This command has been written to simplify the process of retrieving files from AWS glacier.
By using Go you don't need to install aws-cli on your server.

## Usage
you need valid aws creadentials stored in ~/.aws/credentials or in environment variables.

The only command implemented is `getFiles`, you can retrieve the help as follow:
```bash
$ glacier_utils getFiles -h

  Retrieve files from glacier by requesting defrost and download if needed
  
  Usage:
    glacier_utils getFiles [flags]
  
  Flags:
    -b, --bucketName string     name of de bucket to dowload files from
    -r, --bucketRegion string   region where the bucket is located (default "eu-west-1")
        --defrost               defrost files that matches
    -d, --dir string            directory where to save the downloaded files (default ".*")
        --download              download files that matches
    -h, --help                  help for getFiles
    -p, --prefix string         prefix to use to filter result on AWS side ( this increase speed )
    -x, --regexpFilter string   regular expression used to filter files (default ".*")
```
My recommendation is to run the command with `prefix` and `regexpFilter` in order to check the files that will be affected:
```bash
$ glacier_utils getFiles --bucketRegion "eu-west-1" --bucketName "mybucket" --prefix 'date_2020-' --regexpFilter "date_2020-[0-1][0-9]-01/.*"         

#################################################
## Files matching pattern:
#################################################
GLACIER  -  date_2020-01-01/mybackup.bkp
GLACIER  -  date_2020-02-01/mybackup.bkp
GLACIER  -  date_2020-03-01/mybackup.bkp
GLACIER  -  date_2020-04-01/mybackup.bkp
GLACIER  -  date_2020-05-01/mybackup.bkp
GLACIER  -  date_2020-06-01/mybackup.bkp
```
And then run the commad with the `--defrost` and `--download` parameters:
```bash
#################################################
## Files matching pattern:
#################################################
GLACIER  -  date_2020-01-01/mybackup.bkp
GLACIER  -  date_2020-02-01/mybackup.bkp
GLACIER  -  date_2020-03-01/mybackup.bkp
GLACIER  -  date_2020-04-01/mybackup.bkp
GLACIER  -  date_2020-05-01/mybackup.bkp
GLACIER  -  date_2020-06-01/mybackup.bkp

#################################################
## Restoring Objects
#################################################
GLACIER  -  date_2020-01-01/mybackup.bkp
GLACIER  -  date_2020-02-01/mybackup.bkp
GLACIER  -  date_2020-03-01/mybackup.bkp
GLACIER  -  date_2020-04-01/mybackup.bkp
GLACIER  -  date_2020-05-01/mybackup.bkp
GLACIER  -  date_2020-06-01/mybackup.bkp
Skipping download since some item isn't restored yet
```
Please note that the above command does not download files until all restore requests are complete.